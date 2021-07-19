package agent

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// runSocket makes a connection with Slack over WebSocket.
func (s *Server) runSocket(ctx context.Context) error {
	return s.smClient.RunContext(ctx)
}

// ListenInteractiveEvents listens to events from interactive components and
// runs the event handler.
func (s *Server) listenInteractiveEvents(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case envelope, ok := <-s.smClient.Events:
			if !ok {
				return errors.New("channel is closed")
			}

			switch envelope.Type {
			case socketmode.EventTypeInteractive:
				cb, ok := envelope.Data.(slack.InteractionCallback)
				if !ok {
					s.log.Info("received data cannot be converted into slack.InteractionCallback")
					continue
				}

				if isExtendButtonEvent(&cb) {
					namespace, pod, err := getPodFromCallbackEvent(&cb)
					if err != nil {
						return err
					}
					tm, err := getTimeFromCallbackEvent(&cb, time.Now())
					if err != nil {
						return err
					}
					err = s.extendPod(ctx, cb.Channel.ID, namespace, pod, tm)
					if err != nil {
						return err
					}
				}

				s.smClient.Ack(*envelope.Request)

			default:
				s.log.Info("skipped event because type is not "+string(socketmode.EventTypeInteractive),
					"type", envelope.Type,
					"data", envelope.Data,
				)
			}
		}
	}
}

/*
The contents of a callback event is as follows. (some parts are omitted)

- When filtering callback events, check the `action_id`(2) and `block_id`(3).
- The required values for pod extension are (1), (4) and (5).
- This structure is depending on the CI result message. See the `messageCIResult()` function.

---
{
	"type": "block_actions",
	"channel": {
		"id": "<CHANNEL_ID>",                         // (1)
		"name": "<CHANNEL_NAME>"
	},
	"actions": [
		{
			"action_id": "slack-agent-extend-button", // (2)
			"block_id": "slack-agent-extend",         // (3)
			"type": "button",
			"value": "default\/pod"                   // (4)
		}
	],
	"state": {
		"values": {
			"slack-agent-extend": {
				"slack-agent-extend-timepicker": {
					"type": "timepicker",
					"selected_time": "06:47"          // (5)
				}
			}
		}
	}
}
*/

func isExtendButtonEvent(cb *slack.InteractionCallback) bool {
	if cb.Type != slack.InteractionTypeBlockActions {
		return false
	}
	if len(cb.ActionCallback.BlockActions) != 1 {
		return false
	}
	if cb.ActionCallback.BlockActions[0].BlockID != extendBlockID {
		return false
	}
	if cb.ActionCallback.BlockActions[0].ActionID != buttonActionID {
		return false
	}
	return true
}

func getPodFromCallbackEvent(cb *slack.InteractionCallback) (string, string, error) {
	split := strings.Split(cb.ActionCallback.BlockActions[0].Value, "/")
	if len(split) != 2 {
		return "", "", fmt.Errorf("failed to get pod name from callback: %v", cb)
	}
	return split[0], split[1], nil
}

func getTimeFromCallbackEvent(cb *slack.InteractionCallback, baseTime time.Time) (time.Time, error) {
	if _, ok := cb.BlockActionState.Values[extendBlockID]; !ok {
		return time.Time{}, fmt.Errorf("failed to get value from callback: block_id(%s) is not specified", extendBlockID)
	}
	if _, ok := cb.BlockActionState.Values[extendBlockID][pickerActionID]; !ok {
		return time.Time{}, fmt.Errorf("failed to get value from callback: action_id(%s) is not specified", pickerActionID)
	}

	selectedTime := cb.BlockActionState.Values[extendBlockID][pickerActionID].SelectedTime
	split := strings.Split(selectedTime, ":")
	if len(split) != 2 {
		return time.Time{}, fmt.Errorf("invalid time format: %s", selectedTime)
	}
	hh, err := strconv.Atoi(split[0])
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid time format: %s", selectedTime)
	}
	mm, err := strconv.Atoi(split[1])
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid time format: %s", selectedTime)
	}

	return time.Date(baseTime.Year(), baseTime.Month(), baseTime.Day(), hh, mm, 0, 0, baseTime.Location()), nil
}

func (s *Server) extendPod(ctx context.Context, channel, namespace, pod string, tm time.Time) error {
	success := true
	if !s.devMood {
		err := s.updateDeletionTime(ctx, pod, namespace, tm)
		if err != nil {
			s.log.Error(err, "failed to update deletion time",
				"name", pod,
				"namespace", namespace,
				"time", tm,
			)
			success = false
		}
	} else {
		s.log.Info("skip to update deletion time",
			"name", pod,
			"namespace", namespace,
			"time", tm,
		)
		// For testing the slack messages. There is no need to call `rand.Seed()`.
		success = (rand.Intn(2) == 0)
	}

	var msg slack.MsgOption
	if success {
		msg = messagePodExtendSuccess(namespace+"/"+pod, tm)
	} else {
		msg = messagePodExtendFailure(namespace + "/" + pod)
	}
	ctx, cancel := context.WithTimeout(ctx, slackPostTimeout)
	defer cancel()
	_, _, err := s.apiClient.PostMessageContext(ctx, channel, msg)
	if err != nil {
		s.log.Error(err, "failed to send slack message",
			"name", pod,
			"namespace", namespace,
		)
		return err
	}
	return nil
}

func (s *Server) updateDeletionTime(ctx context.Context, namespace, pod string, tm time.Time) error {
	po, err := s.clientset.CoreV1().Pods(namespace).Get(ctx, pod, metav1.GetOptions{})
	if err != nil {
		return err
	}

	return s.runnerClient.PutDeletionTime(ctx, po.Status.PodIP, tm)
}
