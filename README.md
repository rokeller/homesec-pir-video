# homesec-pir-video

Simple golang app for RaspberryPi to read a signal from a PIR sensor and record a video for the duration of the signal.

## What's this

This simple app listens for signal change in the PIR data. When the PIR sensor signals HIGH, the app starts recording (using `raspivid`), and it stops recording when the signal goes back to LOW (or when the app terminates). Videos are created with a resolution of 720x480 and the default h264 encoding at 25 fps. Video files are stored under `/data/video` (make sure the directory exists before starting the app), and the files are named using the timestamp from when the recording started.

You'll likely need to run with `sudo`, since I haven't gotten around to make sure it works as normal user (in the gpio group). You can also run it from root with cron, e.g. with a `crontab` entry:

`@reboot /path/to/homesec-pir-video`

Since raspivid creates raw h264 files, you must play the video with the FPS explicitly set to 25, e.g. with mplayer:

`mplayer -fps 20180211_162256.h264`

## Wiring

Wire the data channel of the PIR sensor to GPIO 4 (pin 7 in the physical numbering, see <https://www.raspberrypi.org/documentation/usage/gpio/>), or change the code accordingly to use a different GPIO pin.
