// devcli implements a command-line interface for the development of Kubernetes applications.
// It provides a simple way to forward local ports to remote ports on Kubernetes pods.
// It also provides a way to forward local ports to remote ports on a bastion server
// that is used to connect to pods. The configuration file contains the name of the
// bastion server, the name of the zone where the bastion server is located, the
// connections that will be forwarded through the bastion server, and the rules for
// port forwarding.

// The rules are used to automatically forward local ports to remote ports on pods.
// The rules are applied to pods that match the namespace and app labels.
// The rules are applied in the order that they are specified in the configuration file.
// If a pod matches multiple rules, the first rule that matches is used.
// If a pod matches no rules, the pod is ignored.

// A context is created that will be used to cancel the port-forward commands
// when the program is interrupted. The program exits with an error if
// the context is canceled.
// SIGINT and SIGTERM signals are listened for.
// When the program is interrupted, the context is canceled and the program waits
// for the port-forward commands to gracefully terminate.

// The program is interrupted when the user presses Ctrl+C.
// The program waits for the port-forward commands to gracefully terminate before exiting.
//  The program exits immediately if the user presses Ctrl+C a second time.
// The program exits with an error if the configuration file is not specified,
// if the configuration file cannot be read or parsed, if there are duplicate local ports,
// or if kubectl is not installed or not in the system's PATH.
// The program exits with an error if the port-forward command fails.
// The program exits with an error if the bastion server connection fails.

package main
