#!/usr/bin/env node
// NOTE: This script is copied from https://github.com/actions/runner/blob/main/src/Misc/layoutbin/RunnerService.js and is updated in some lines of code.
// Copyright (c) GitHub. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

var childProcess = require("child_process");
var path = require("path")

var supported = ['linux', 'darwin']

if (supported.indexOf(process.platform) == -1) {
    console.log('Unsupported platform: ' + process.platform);
    console.log('Supported platforms are: ' + supported.toString());
    process.exit(1);
}

var stopping = false;
var listener = null;

var runService = function() {
    var listenerExePath = path.join(__dirname, '../bin/Runner.Listener');
    var interactive = process.argv[2] === "interactive";

    if(!stopping) {
        try {
            if (interactive) {
                console.log('Starting Runner listener interactively');
                // NOTE: --once is added to kill runner after job finishes
                listener = childProcess.spawn(listenerExePath, ['run', '--once'], { env: process.env });
            } else {
                console.log('Starting Runner listener with startup type: service');
                // NOTE: --once is added to kill runner after job finishes
                listener = childProcess.spawn(listenerExePath, ['run', '--startuptype', 'service', '--once'], { env: process.env });
            }

            console.log('Started listener process');

            listener.stdout.on('data', (data) => {
                process.stdout.write(data.toString('utf8'));
            });

            listener.stderr.on('data', (data) => {
                process.stdout.write(data.toString('utf8'));
            });

            listener.on('close', (code) => {
                console.log(`Runner listener exited with error code ${code}`);

                if (code === 0) {
                    console.log('Runner listener exit with 0 return code, stop the service, no retry needed.');
                    stopping = true;
                } else if (code === 1) {
                    console.log('Runner listener exit with terminated error, stop the service, no retry needed.');
                    stopping = true;
                } else if (code === 2) {
                    console.log('Runner listener exit with retryable error, re-launch runner in 5 seconds.');
                } else if (code === 3) {
                    console.log('Runner listener exit because of updating, re-launch runner in 5 seconds.');
                } else {
                    console.log('Runner listener exit with undefined return code, re-launch runner in 5 seconds.');
                }

                if(!stopping) {
                    setTimeout(runService, 5000);
                }
            });

        } catch(ex) {
            console.log(ex);
        }
    }
}

runService();
console.log('Started running service');

var gracefulShutdown = function(code) {
    console.log('Shutting down runner listener');
    stopping = true;
    if (listener) {
        console.log('Sending SIGINT to runner listener to stop');
        listener.kill('SIGINT');

        // TODO wait for 30 seconds and send a SIGKILL
    }
}

process.on('SIGINT', () => {
    gracefulShutdown(0);
});

process.on('SIGTERM', () => {
    gracefulShutdown(0);
});
