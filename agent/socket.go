package agent

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"
	corev1 "k8s.io/api/core/v1"
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
					err = s.extendPod(ctx, cb.Channel.ID, namespace, pod)
					if err != nil {
						return err
					}
				} else if isDeleteButtonEvent(&cb) {
					namespace, pod, err := getPodFromCallbackEvent(&cb)
					if err != nil {
						return err
					}
					err = s.deletePod(ctx, cb.Channel.ID, namespace, pod)
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
The contents of a callback event is as follows. (some parts are omitted - see https://api.slack.com/reference/interaction-payloads/block-actions for more details)

- When filtering callback events, check the `action_id`(2) and `block_id`(3).
- The required values for pod extension are (1) and (4).
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
	]
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
	if cb.ActionCallback.BlockActions[0].ActionID != extendButtonActionID {
		return false
	}
	return true
}

func isDeleteButtonEvent(cb *slack.InteractionCallback) bool {
	if cb.Type != slack.InteractionTypeBlockActions {
		return false
	}
	if len(cb.ActionCallback.BlockActions) != 1 {
		return false
	}
	if cb.ActionCallback.BlockActions[0].BlockID != deleteBlockID {
		return false
	}
	if cb.ActionCallback.BlockActions[0].ActionID != deleteButtonActionID {
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

func (s *Server) extendPod(ctx context.Context, channel, namespace, pod string) error {
	po, err := s.clientset.CoreV1().Pods(namespace).Get(ctx, pod, metav1.GetOptions{})
	if err != nil {
		s.log.Error(err, "failed to get pod info",
			"name", pod,
			"namespace", namespace,
		)
		return err
	}

	status, err := s.runnerClient.GetStatus(ctx, po.Status.PodIP)
	if err != nil {
		s.log.Error(err, "failed to get runner status",
			"name", pod,
			"namespace", namespace,
		)
		return err
	}
	var tm time.Time
	if status.DeletionTime != nil {
		tm = *status.DeletionTime
	} else {
		tm = time.Now().UTC() // We now expect DeletionTime is not nil at this point. This line is a fail safe.
		s.log.Info("deletion time was not set beforehand",
			"name", pod,
			"namespace", namespace,
		)
	}
	tm = tm.Add(time.Hour * extendDurationHours)
	limit := time.Now().UTC().Add(time.Hour * extendLimitHours)
	if tm.After(limit) {
		tm = limit
	}

	return s.putDeletionTime(ctx, channel, namespace, pod, po, tm)
}

func (s *Server) deletePod(ctx context.Context, channel, namespace, pod string) error {
	po, err := s.clientset.CoreV1().Pods(namespace).Get(ctx, pod, metav1.GetOptions{})
	if err != nil {
		s.log.Error(err, "failed to get pod info",
			"name", pod,
			"namespace", namespace,
		)
		return err
	}

	return s.putDeletionTime(ctx, channel, namespace, pod, po, time.Time{})
}

func (s *Server) putDeletionTime(ctx context.Context, channel, namespace, pod string, po *corev1.Pod, tm time.Time) error {
	success := true
	if !s.devMood {
		err := s.runnerClient.PutDeletionTime(ctx, po.Status.PodIP, tm)
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
