/*
Copyright 2021 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package process

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sync"
	"syscall"
	"time"

	"github.com/platform-mesh/golang-commons/logger"
	"gopkg.in/yaml.v3"
)

// ListenAddr represents some listening address and port.
type ListenAddr struct {
	Address string
	Port    string
}

// URL returns a URL for this address with the given scheme and subpath.
func (l *ListenAddr) URL(scheme string, path string) *url.URL {
	return &url.URL{
		Scheme: scheme,
		Host:   l.HostPort(),
		Path:   path,
	}
}

// HostPort returns the joined host-port pair for this address.
func (l *ListenAddr) HostPort() string {
	return net.JoinHostPort(l.Address, l.Port)
}

// HealthCheck describes the information needed to health-check a process via
// some health-check URL.
type HealthCheck struct {
	url.URL
	KcpAssetPath string

	// HealthCheckPollInterval is the interval which will be used for polling the
	// endpoint described by Host, Port, and Path.
	//
	// If left empty it will default to 100 Milliseconds.
	PollInterval time.Duration
}

// State define the state of the process.
type State struct {
	Cmd *exec.Cmd

	// HealthCheck describes how to check if this process is up.  If we get an http.StatusOK,
	// we assume the process is ready to operate.
	//
	// For example, the /healthz endpoint of the k8s API server, or the /health endpoint of etcd.
	HealthCheck HealthCheck

	Args []string

	StopTimeout  time.Duration
	StartTimeout time.Duration

	Dir              string
	DirNeedsCleaning bool
	Path             string

	// ready holds whether the process is currently in ready state (hit the ready condition) or not.
	// It will be set to true on a successful `Start()` and set to false on a successful `Stop()`
	ready bool

	// waitDone is closed when our call to wait finishes up, and indicates that
	// our process has terminated.
	waitDone chan struct{}
	errMu    sync.Mutex
	exitErr  error
	exited   bool
}

// Init sets up this process, configuring binary paths if missing, initializing
// temporary directories, etc.
//
// This defaults all defaultable fields.
func (ps *State) Init(name string) error {
	if ps.Path == "" {
		if name == "" {
			return fmt.Errorf("must have at least one of name or path")
		}
		ps.Path = BinPathFinder(name, "")
	}

	if ps.Dir == "" {
		newDir, err := os.MkdirTemp("", "k8s_test_framework_")
		if err != nil {
			return err
		}
		ps.Dir = newDir
		ps.DirNeedsCleaning = true
	}

	if ps.StartTimeout == 0 {
		ps.StartTimeout = 20 * time.Second
	}

	if ps.StopTimeout == 0 {
		ps.StopTimeout = 20 * time.Second
	}
	return nil
}

type stopChannel chan struct{}

// CheckFlag checks the help output of this command for the presence of the given flag, specified
// without the leading `--` (e.g. `CheckFlag("insecure-port")` checks for `--insecure-port`),
// returning true if the flag is present.
func (ps *State) CheckFlag(flag string) (bool, error) {
	cmd := exec.Command(ps.Path, "--help")
	outContents, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("unable to run command %q to check for flag %q: %w", ps.Path, flag, err)
	}
	pat := `(?m)^\s*--` + flag + `\b` // (m --> multi-line --> ^ matches start of line)
	matched, err := regexp.Match(pat, outContents)
	if err != nil {
		return false, fmt.Errorf("unable to check command %q for flag %q in help output: %w", ps.Path, flag, err)
	}
	return matched, nil
}

// Start starts the apiserver, waits for it to come up, and returns an error,
// if occurred.
func (ps *State) Start(stdout, stderr io.Writer, log *logger.Logger) (err error) {
	if ps.ready {
		return nil
	}

	ps.Cmd = exec.Command(ps.Path, ps.Args...)
	ps.Cmd.Dir = ps.Dir
	ps.Cmd.Stdout = stdout
	ps.Cmd.Stderr = stderr
	ps.Cmd.SysProcAttr = GetSysProcAttr()

	ready := make(chan bool)
	timedOut := time.After(ps.StartTimeout)
	pollerStopCh := make(stopChannel)
	go pollURLUntilOK(ps.HealthCheck.URL, ps.HealthCheck.PollInterval, ps.HealthCheck.KcpAssetPath, ready, pollerStopCh, log)

	ps.waitDone = make(chan struct{})

	if err := ps.Cmd.Start(); err != nil {
		ps.errMu.Lock()
		defer ps.errMu.Unlock()
		ps.exited = true
		return err
	}
	go func() {
		defer close(ps.waitDone)
		err := ps.Cmd.Wait()

		ps.errMu.Lock()
		defer ps.errMu.Unlock()
		ps.exitErr = err
		ps.exited = true
	}()

	select {
	case <-ready:
		ps.ready = true
		return nil
	case <-ps.waitDone:
		close(pollerStopCh)
		return fmt.Errorf("timeout waiting for process %s to start successfully "+
			"(it may have failed to start, or stopped unexpectedly before becoming ready)",
			path.Base(ps.Path))
	case <-timedOut:
		close(pollerStopCh)
		if ps.Cmd != nil {
			// intentionally ignore this -- we might've crashed, failed to start, etc
			ps.Cmd.Process.Signal(syscall.SIGTERM) //nolint:errcheck
		}
		return fmt.Errorf("timeout waiting for process %s to start", path.Base(ps.Path))
	}
}

// Exited returns true if the process exited, and may also
// return an error (as per Cmd.Wait) if the process did not
// exit with error code 0.
func (ps *State) Exited() (bool, error) {
	ps.errMu.Lock()
	defer ps.errMu.Unlock()
	return ps.exited, ps.exitErr
}

func pollURLUntilOK(url url.URL, interval time.Duration, kcpAssetPath string, ready chan bool, stopCh stopChannel, log *logger.Logger) {

	if interval <= 0 {
		interval = 5000 * time.Millisecond
	}
	for {
		token, ca, err := readTokenAndCA(kcpAssetPath)
		if err != nil {
			log.Info().Msg("health check failed. Credentials not ready")
			time.Sleep(interval)
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(ca)
		client := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs: caCertPool,
				},
			},
		}
		req, err := http.NewRequest(http.MethodGet, url.String(), nil)
		if err != nil {
			log.Fatal().Err(err).Msg("error creating request")
		}
		if token != "" {
			req.Header.Add("Authorization", "Bearer "+token)
		}
		res, err := client.Do(req)
		if err == nil {
			if err != nil {
				fmt.Println("Error reading response body:", err)
				return
			}
			err := res.Body.Close()
			if err != nil {
				fmt.Println("Error closing response body:", err)
				return
			}
			if res.StatusCode == http.StatusOK {
				log.Info().Int("status", res.StatusCode).Msg("KCP Ready (health check succeeded)")
				ready <- true
				return
			}
			log.Info().Int("status", res.StatusCode).Msg("Waiting for KCP to get ready (health check failed)")
		}

		select {
		case <-stopCh:
			return
		default:
			time.Sleep(interval)
		}
	}
}

type kubeconfig struct {
	Users []struct {
		Name string `yaml:"name"`
		User struct {
			Token string `yaml:"token"`
		} `yaml:"user"`
	}
}

func readTokenAndCA(path string) (string, []byte, error) {
	adminKubeconfigPath := filepath.Join(path, "admin.kubeconfig")
	// check if file exists
	if _, err := os.Stat(adminKubeconfigPath); os.IsNotExist(err) {
		return "", nil, fmt.Errorf("file %s does not exist", adminKubeconfigPath)
	}
	file, err := os.Open(adminKubeconfigPath)
	if err != nil {
		return "", nil, fmt.Errorf("error opening file %s: %w", path, err)
	}
	defer file.Close() //nolint:errcheck

	data, err := io.ReadAll(file)
	if err != nil {
		return "", nil, fmt.Errorf("error reading file %s: %w", path, err)
	}

	var config kubeconfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return "", nil, fmt.Errorf("error unmarshalling yaml from file %s: %w", path, err)
	}

	var userToken string
	for _, user := range config.Users {
		if user.Name == "kcp-admin" {
			userToken = user.User.Token
		}
	}
	if userToken == "" {
		return "", nil, fmt.Errorf("token not found in kubeconfig file %s", path)
	}

	certPath := filepath.Join(path, "apiserver.crt")
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		return "", nil, fmt.Errorf("file %s does not exist", certPath)
	}
	file, err = os.Open(certPath)
	if err != nil {
		return "", nil, fmt.Errorf("error opening file %s: %w", path, err)
	}
	defer file.Close() //nolint:errcheck

	data, err = io.ReadAll(file)
	if err != nil {
		return "", nil, fmt.Errorf("error reading file %s: %w", path, err)
	}

	return userToken, data, nil
}

// Stop stops this process gracefully, waits for its termination, and cleans up
// the CertDir if necessary.
func (ps *State) Stop() error {
	// Always clear the directory if we need to.
	defer func() {
		if ps.DirNeedsCleaning {
			_ = os.RemoveAll(ps.Dir)
		}
	}()
	if ps.Cmd == nil {
		return nil
	}
	if done, _ := ps.Exited(); done {
		return nil
	}
	if err := ps.Cmd.Process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("unable to signal for process %s to stop: %w", ps.Path, err)
	}

	timedOut := time.After(ps.StopTimeout)

	select {
	case <-ps.waitDone:
		break
	case <-timedOut:
		if err := ps.Cmd.Process.Signal(syscall.SIGKILL); err != nil {
			return fmt.Errorf("unable to kill process %s: %w", ps.Path, err)
		}
		return fmt.Errorf("timeout waiting for process %s to stop", path.Base(ps.Path))
	}
	ps.ready = false
	return nil
}
