package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"reflect"

	"github.com/bgentry/speakeasy"
	"github.com/spf13/cobra"

	"github.com/rivine/rivine/api"
	"github.com/rivine/rivine/build"
)

// flags
var (
	addr         string // override default API address
	initPassword bool   // supply a custom password when creating a wallet
)

// exit codes
// inspired by sysexits.h
const (
	exitCodeGeneral = 1  // Not in sysexits.h, but is standard practice.
	exitCodeUsage   = 64 // EX_USAGE in sysexits.h
)

// non2xx returns true for non-success HTTP status codes.
func non2xx(code int) bool {
	return code < 200 || code > 299
}

// decodeError returns the api.Error from a API response. This method should
// only be called if the response's status code is non-2xx. The error returned
// may not be of type api.Error in the event of an error unmarshalling the
// JSON.
func decodeError(resp *http.Response) error {
	var apiErr api.Error
	err := json.NewDecoder(resp.Body).Decode(&apiErr)
	if err != nil {
		return err
	}
	return apiErr
}

// apiGet wraps a GET request with a status code check, such that if the GET does
// not return 2xx, the error will be read and returned. The response body is
// not closed.
func apiGet(call string) (*http.Response, error) {
	if host, port, _ := net.SplitHostPort(addr); host == "" {
		addr = net.JoinHostPort("localhost", port)
	}
	resp, err := api.HttpGET("http://" + addr + call)
	if err != nil {
		return nil, errors.New("no response from daemon")
	}
	// check error code
	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()
		// Prompt for password and retry request with authentication.
		password, err := speakeasy.Ask("API password: ")
		if err != nil {
			return nil, err
		}
		resp, err = api.HttpGETAuthenticated("http://"+addr+call, password)
		if err != nil {
			return nil, errors.New("no response from daemon - authentication failed")
		}
	}
	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		return nil, errors.New("API call not recognized: " + call)
	}
	if non2xx(resp.StatusCode) {
		err := decodeError(resp)
		resp.Body.Close()
		return nil, err
	}
	return resp, nil
}

// getAPI makes a GET API call and decodes the response. An error is returned
// if the response status is not 2xx.
func getAPI(call string, obj interface{}) error {
	resp, err := apiGet(call)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return errors.New("expecting a response, but API returned status code 204 No Content")
	}

	err = json.NewDecoder(resp.Body).Decode(obj)
	if err != nil {
		return err
	}
	return nil
}

// get makes an API call and discards the response. An error is returned if the
// response status is not 2xx.
func get(call string) error {
	resp, err := apiGet(call)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// apiPost wraps a POST request with a status code check, such that if the POST
// does not return 2xx, the error will be read and returned. The response body
// is not closed.
func apiPost(call, vals string) (*http.Response, error) {
	if host, port, _ := net.SplitHostPort(addr); host == "" {
		addr = net.JoinHostPort("localhost", port)
	}

	resp, err := api.HttpPOST("http://"+addr+call, vals)
	if err != nil {
		return nil, errors.New("no response from daemon")
	}
	// check error code
	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()
		// Prompt for password and retry request with authentication.
		password, err := speakeasy.Ask("API password: ")
		if err != nil {
			return nil, err
		}
		resp, err = api.HttpPOSTAuthenticated("http://"+addr+call, vals, password)
		if err != nil {
			return nil, errors.New("no response from daemon - authentication failed")
		}
	}
	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		return nil, errors.New("API call not recognized: " + call)
	}
	if non2xx(resp.StatusCode) {
		err := decodeError(resp)
		resp.Body.Close()
		return nil, err
	}
	return resp, nil
}

// postResp makes a POST API call and decodes the response. An error is
// returned if the response status is not 2xx.
func postResp(call, vals string, obj interface{}) error {
	resp, err := apiPost(call, vals)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return errors.New("expecting a response, but API returned status code 204 No Content")
	}

	err = json.NewDecoder(resp.Body).Decode(obj)
	if err != nil {
		return err
	}
	return nil
}

// post makes an API call and discards the response. An error is returned if
// the response status is not 2xx.
func post(call, vals string) error {
	resp, err := apiPost(call, vals)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// wrap wraps a generic command with a check that the command has been
// passed the correct number of arguments. The command must take only strings
// as arguments.
func wrap(fn interface{}) func(*cobra.Command, []string) {
	fnVal, fnType := reflect.ValueOf(fn), reflect.TypeOf(fn)
	if fnType.Kind() != reflect.Func {
		panic("wrapped function has wrong type signature")
	}
	for i := 0; i < fnType.NumIn(); i++ {
		if fnType.In(i).Kind() != reflect.String {
			panic("wrapped function has wrong type signature")
		}
	}

	return func(cmd *cobra.Command, args []string) {
		if len(args) != fnType.NumIn() {
			cmd.UsageFunc()(cmd)
			os.Exit(exitCodeUsage)
		}
		argVals := make([]reflect.Value, fnType.NumIn())
		for i := range args {
			argVals[i] = reflect.ValueOf(args[i])
		}
		fnVal.Call(argVals)
	}
}

// die prints its arguments to stderr, then exits the program with the default
// error code.
func die(args ...interface{}) {
	fmt.Fprintln(os.Stderr, args...)
	os.Exit(exitCodeGeneral)
}

func version() {
	println("Rivine Client v" + build.Version.String())
}

func main() {
	root := &cobra.Command{
		Use:   os.Args[0],
		Short: "Rivine Client v" + build.Version.String(),
		Long:  "Rivine Client v" + build.Version.String(),
		Run:   wrap(consensuscmd),
	}

	// create command tree
	root.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Long:  "Print version information.",
		Run:   wrap(version),
	})

	root.AddCommand(stopCmd)

	root.AddCommand(updateCmd)
	updateCmd.AddCommand(updateCheckCmd)

	root.AddCommand(walletCmd)
	walletCmd.AddCommand(walletAddressCmd, walletAddressesCmd, walletInitCmd,
		walletLoadCmd, walletLockCmd, walletSeedsCmd, walletSendCmd,
		walletBalanceCmd, walletTransactionsCmd, walletUnlockCmd, walletBlockStakeStatCmd)
	walletInitCmd.Flags().BoolVarP(&initPassword, "password", "p", false, "Prompt for a custom password")
	walletSendCmd.AddCommand(walletSendSiacoinsCmd, walletSendSiafundsCmd)
	walletLoadCmd.AddCommand(walletLoadSeedCmd)

	root.AddCommand(gatewayCmd)
	gatewayCmd.AddCommand(gatewayConnectCmd, gatewayDisconnectCmd, gatewayAddressCmd, gatewayListCmd)

	root.AddCommand(consensusCmd)

	// parse flags
	root.PersistentFlags().StringVarP(&addr, "addr", "a", "localhost:23110", "which host/port to communicate with (i.e. the host/port rivined is listening on)")

	// run
	if err := root.Execute(); err != nil {
		// Since no commands return errors (all commands set Command.Run instead of
		// Command.RunE), Command.Execute() should only return an error on an
		// invalid command or flag. Therefore Command.Usage() was called (assuming
		// Command.SilenceUsage is false) and we should exit with exitCodeUsage.
		os.Exit(exitCodeUsage)
	}
}
