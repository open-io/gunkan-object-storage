//
// Copyright 2019 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

package kv

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// Once the hook is called, the <pathSocket> and <pathDirectory> placeholders have been
// set with the actual path of the service endpoint,
func kvServerFixture(pathSocket, pathDirectory *string, t *testing.T, cb func(*testing.T)) {
	var err error
	var pathWorking string
	deadline := time.Now().Add(1 * time.Minute)

	pathWorking, err = ioutil.TempDir("", "kv-")
	if err != nil {
		t.Fatalf("Failed to create the KV file marker")
	}

	p := fmt.Sprintf("%s/sock", pathWorking)
	*pathSocket = "ipc://" + p
	*pathDirectory = fmt.Sprintf("%s/vol", pathWorking)
	args := []string{"gunkan-kv", *pathSocket, *pathDirectory}

	t.Log("KV starting: ", strings.Join(args, " "))

	err = os.MkdirAll(*pathDirectory, 0755)
	if err != nil {
		t.Fatalf("Failed to create the KV base directory")
	}
	//defer os.RemoveAll(pathWorking)

	err = os.Chdir(pathWorking)
	if err != nil {
		t.Fatalf("Failed to chdir in the working directory")
	}

	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	if err := cmd.Start(); err != nil {
		t.Fatalf("KV start error: %s", err.Error())
	}
	defer func() { cmd.Process.Kill(); cmd.Wait() }()

	time.Sleep(10 * time.Millisecond)
	for {
		cnx, err := net.DialTimeout("unix", p, 1*time.Second)
		if err != nil {
			if time.Now().After(deadline) {
				t.Fatalf("KV connect error: %s", err.Error())
			} else {
				t.Logf("KV not started yet: %s", err.Error())
				time.Sleep(100 * time.Millisecond)
			}
		} else {
			cnx.Close()
			t.Logf("KV started and ready")
			break
		}
	}

	cb(t)
}

func TestApi_cycle(t *testing.T) {
	var pathSocket, pathVolume string
	kvServerFixture(&pathSocket, &pathVolume, t, func(t *testing.T) {
		for i := 0; i < 7; i++ {
			client, err := Dial(pathSocket)
			t.Logf("Dial(%s): %v", pathSocket, err)
			if err != nil {
				t.Fatal("Dial error to ", pathSocket)
			} else {
				t.Log("Connected to ", pathSocket)
				client.Close()
			}
		}
	})
}
