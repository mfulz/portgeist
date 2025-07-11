// Package launchbackends implements a cgroup-based launcher that
// starts a background proxy daemon (e.g. redsocks) via binaryBackend
// and wraps the actual user command inside a systemd-run cgroup slice.
package launchbackends

type cgroupBackend struct{}

func init() {
	// ilauncher.RegisterBackend(&cgroupBackend{})
}

func (c *cgroupBackend) Method() string {
	return "cgroup"
}

// func (c *cgroupBackend) GetInstance(name string, cfg ilauncher.FileConfig, host string, port int, ctx ilauncher.Context) (*cobra.Command, error) {
// 	cmd := &cobra.Command{
// 		Use:   name,
// 		Short: "Launch a process in systemd cgroup slice after starting proxy daemon",
// 		RunE: func(cmd *cobra.Command, args []string) error {
// 			if len(args) == 0 {
// 				fmt.Fprintln(os.Stderr, "Missing command to execute.")
// 				os.Exit(1)
// 			}
//
// 			// Start background proxy via binary backend
// 			proxyBackend, err := ilauncher.GetBackend("binary")
// 			if err != nil {
// 				return fmt.Errorf("binary backend not found: %w", err)
// 			}
//
// 			daemonCmd, err := proxyBackend.GetInstance("proxy-helper-"+name, cfg, host, port, ctx)
// 			if err != nil {
// 				return fmt.Errorf("failed to init proxy helper: %w", err)
// 			}
//
// 			go func() {
// 				_ = daemonCmd.Execute()
// 			}()
//
// 			// Wait a moment to ensure proxy daemon is up (optional, can be improved)
// 			time.Sleep(500 * time.Millisecond)
//
// 			// Generate systemd-run slice name
// 			shortID := uuid.New().String()[:8]
// 			sliceName := fmt.Sprintf("portgeist-%s.slice", shortID)
//
// 			// Build final command inside systemd-run
// 			runArgs := []string{
// 				"--slice=" + sliceName,
// 				"--unit=pg-run-" + shortID,
// 				"--quiet",
// 			}
// 			runArgs = append(runArgs, args...)
//
// 			runCmd := exec.Command("systemd-run", runArgs...)
// 			runCmd.Stdout = os.Stdout
// 			runCmd.Stderr = os.Stderr
// 			runCmd.Stdin = os.Stdin
// 			runCmd.Env = os.Environ()
//
// 			return runCmd.Run()
// 		},
// 	}
//
// 	return cmd, nil
// }
