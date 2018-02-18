package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

type Direction int

const (
	In Direction = iota
	Out
)

type Pin struct {
	Id      int
	Dir     Direction
	pinPath string
	fdDir   *os.File
}

func export(id int) {
	err := ioutil.WriteFile("/sys/class/gpio/export", []byte(strconv.Itoa(id)), 0666)

	if nil != err {
		log.Panicf("Failed to export Pin %d: %v", id, err)
	}
}

func unexport(id int) {
	err := ioutil.WriteFile("/sys/class/gpio/unexport", []byte(strconv.Itoa(id)), 0666)

	if nil != err {
		log.Panicf("Failed to unexport Pin %d: %v", id, err)
	}
}

func OpenPin(id int, dir Direction) *Pin {
	export(id)

	pinPath := fmt.Sprintf("/sys/class/gpio/gpio%d", id)
	fdDir, err := os.OpenFile(pinPath+"/direction", os.O_WRONLY|os.O_SYNC, 0666)

	if nil != err {
		log.Panicf("Failed to open direction file for Pin %d: %v", id, err)
	}

	var dirString string

	switch dir {
	case In:
		dirString = "in"

	case Out:
		dirString = "out"
	}

	_, err = fdDir.Write([]byte(dirString))

	if nil != err {
		log.Panicf("Failed to set direction of Pin %d to '%s': %v", id, dirString, err)
	}

	return &Pin{
		Id:      id,
		Dir:     dir,
		pinPath: pinPath,
		fdDir:   fdDir,
	}
}

func (pin *Pin) Read() bool {
	fdValue, err := os.OpenFile(pin.pinPath+"/value", os.O_RDONLY|os.O_SYNC, 0666)

	if nil != err {
		log.Panicf("Failed to open value file for Pin %d: %v", pin.Id, err)
	}

	defer fdValue.Close()

	buf := make([]byte, 4)

	for {
		read, err := fdValue.Read(buf)

		if nil != err {
			if io.EOF != err {
				log.Panicf("Failed to read from Pin %d: %v", pin.Id, err)
			}
		} else if 1 <= read {
			break
		}
	}

	return buf[0] == '1'
}

func (pin *Pin) Close() {
	pin.fdDir.Close()

	unexport(pin.Id)
}

func startRecording() *exec.Cmd {
	timestamp := time.Now()
	baseName := fmt.Sprintf("/data/video/%s", timestamp.Format("20060102_150405"))
	path := baseName + ".h264"
	cmd := exec.Command("raspivid", "-t", "0", "-o", path, "-w", "720", "-h", "480", "-fps", "25", "-b", "250000")
	cmd.Start()

	srt := fmt.Sprintf("1\n00:00:00,000 --> 00:01:00,000\n%s\n", timestamp.Format("2006-01-02 15:04:05"))
	ioutil.WriteFile(baseName+".srt", []byte(srt), 0644)

	return cmd
}

func recorder(commands <-chan string) {
	var cmdRecord *exec.Cmd = nil

	for {
		select {
		case cmd := <-commands:
			switch cmd {
			case "start":
				log.Println("Start recording.")
				cmdRecord = startRecording()

			case "stop":
				log.Println("Stop recording.")

				if nil != cmdRecord {
					cmdRecord.Process.Signal(syscall.SIGINT)
				}

			default:
				log.Printf("Unsupported command: %s", cmd)
			}
		}
	}
}

func checkPIR(pin *Pin, signals <-chan os.Signal, done chan<- bool, commands chan<- string) {
	lastHigh := false

	for {
		select {
		case sig := <-signals:
			log.Printf("Got signal: %v", sig)

			// If we're currently recording, stop the recording.
			if lastHigh {
				commands <- "stop"
			}

			done <- true
			break

		default:
			curHigh := pin.Read()

			if curHigh != lastHigh {
				if lastHigh {
					commands <- "stop"
				} else {
					commands <- "start"
				}

				lastHigh = curHigh
			}

			time.Sleep(time.Millisecond * 250)
		}
	}
}

func main() {
	signals := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	commands := make(chan string, 1)

	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("PIR-controlled Video Recording Controller")

	pirIn := OpenPin(4, In)
	defer pirIn.Close()

	go recorder(commands)
	go checkPIR(pirIn, signals, done, commands)

	<-done
}
