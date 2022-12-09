// Copyright © 2022 Kaleido, Inc.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func waitAndDrain(c chan bool) {
	<-c
	for {
		select {
		case <-c:
		default:
			return
		}
	}
}

func TestFileListenerE2E(t *testing.T) {

	logrus.SetLevel(logrus.DebugLevel)
	tmpDir := t.TempDir()

	viper.SetConfigType("yaml")
	viper.SetConfigFile(fmt.Sprintf("%s/test.yaml", tmpDir))

	// Start listener on empty dir
	fsListenerDone := make(chan struct{})
	fsListenerFired := make(chan bool, 5)
	ctx, cancelCtx := context.WithCancel(context.Background())
	err := WatchConfig(ctx, func() {
		err := viper.ReadInConfig()
		assert.NoError(t, err)
		fsListenerFired <- true
	}, func() {
		close(fsListenerDone)
	})
	assert.NoError(t, err)

	// Create the file
	os.WriteFile(fmt.Sprintf("%s/test.yaml", tmpDir), []byte(`{"ut_conf": "one"}`), 0664)
	waitAndDrain(fsListenerFired)
	assert.Equal(t, "one", viper.Get("ut_conf"))

	// Write an update
	os.WriteFile(fmt.Sprintf("%s/test.yaml", tmpDir), []byte(`{"ut_conf": "two"}`), 0664)
	waitAndDrain(fsListenerFired)
	assert.Equal(t, "two", viper.Get("ut_conf"))

	// Rename in another file
	os.WriteFile(fmt.Sprintf("%s/another.yaml", tmpDir), []byte(`{"ut_conf": "three"}`), 0664)
	os.Rename(fmt.Sprintf("%s/another.yaml", tmpDir), fmt.Sprintf("%s/test.yaml", tmpDir))
	waitAndDrain(fsListenerFired)
	assert.Equal(t, "three", viper.Get("ut_conf"))

	// Delete and recreate
	os.Remove(fmt.Sprintf("%s/test.yaml", tmpDir))
	os.WriteFile(fmt.Sprintf("%s/test.yaml", tmpDir), []byte(`{"ut_conf": "four"}`), 0664)
	waitAndDrain(fsListenerFired)
	assert.Equal(t, "four", viper.Get("ut_conf"))

	defer func() {
		cancelCtx()
		if a := recover(); a != nil {
			panic(a)
		}
		<-fsListenerDone
	}()

}

func TestFileListenerFail(t *testing.T) {

	logrus.SetLevel(logrus.DebugLevel)
	tmpDir := t.TempDir()
	os.RemoveAll(tmpDir)

	viper.SetConfigType("yaml")
	viper.SetConfigFile(fmt.Sprintf("%s/test.yaml", tmpDir))

	err := WatchConfig(context.Background(), nil, nil)
	assert.Regexp(t, "FF00194", err)
}

func TestFileListenerLogError(t *testing.T) {

	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()
	errors := make(chan error)
	fsListenerDone := make(chan struct{})
	go fsListenerLoop(ctx, "somefile", func() {}, func() { close(fsListenerDone) }, make(chan fsnotify.Event), errors)

	errors <- fmt.Errorf("pop")
	cancelCtx()
	<-fsListenerDone
}
