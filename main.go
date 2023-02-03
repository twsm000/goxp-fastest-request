package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const (
	timeoutUsage = `timeout define a limit to make all requests. Examples 300ms, -1.5h or "2h45m".
Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".`
)

var (
	ErrInvalidFlags   = errors.New("failed to parse flags")
	ErrInvalidCEP     = errors.New("invalid cep")
	ErrInvalidTimeout = errors.New("invalid timeout")
)

func main() {
	flags, usage, err := ParseCLIFlags(os.Args[0], os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(1)
	}

	resp, err := GetCEP(flags.cep, flags.timeout)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stdout, "%+v\n", resp)
}

type CLIFlags struct {
	cep     string
	timeout time.Duration
}

func ParseCLIFlags(progname string, args []string) (cliFlags *CLIFlags, usage string, err error) {
	var cep string
	var timeoutStr string
	var buf bytes.Buffer

	flags := flag.NewFlagSet(progname, flag.ContinueOnError)
	flags.SetOutput(&buf)
	flags.StringVar(&cep, "cep", "", "make a cep request")
	flags.StringVar(&timeoutStr, "timeout", "1s", timeoutUsage)

	err = flags.Parse(args)
	if buf.Len() == 0 {
		flags.PrintDefaults()
	}
	usage = buf.String()

	if err != nil {
		err = fmt.Errorf("%w: %w", ErrInvalidFlags, err)
		return
	}

	if cep == "" {
		err = ErrInvalidCEP
		return
	}

	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		err = ErrInvalidTimeout
		return
	}

	cliFlags = &CLIFlags{
		cep:     cep,
		timeout: timeout,
	}
	return
}

func GetCEP(cep string, timeout time.Duration) (*Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	urls := []string{
		"https://cdn.apicep.com/file/apicep/" + cep + ".json",
		"http://viacep.com.br/ws/" + cep + "/json/",
	}

	requestsReponseStream := MakeRequests(ctx, urls)

	var respReceived int
	var errs []error
	for respReceived < len(urls) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()

		case result := <-requestsReponseStream:
			respReceived++
			if result.err != nil {
				errs = append(errs, result.err)
				continue
			}

			return result.resp, nil
		}
	}

	return nil, errors.Join(errs...)
}

type Response struct {
	URL  string `json:"url"`
	Data string `json:"data"`
}

type RequestResult struct {
	err  error
	resp *Response
}

func MakeRequests(ctx context.Context, urls []string) <-chan *RequestResult {
	stream := make(chan *RequestResult)

	for _, url := range urls {
		go makeRequest(ctx, stream, url)
	}

	return stream
}

func makeRequest(ctx context.Context, sender chan<- *RequestResult, url string) {
	result := &RequestResult{}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		result.err = err
		SendData(ctx, sender, result)
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		result.err = err
		SendData(ctx, sender, result)
		return
	}
	defer resp.Body.Close()

	select {
	case <-ctx.Done():
		return

	default:
		buf := new(bytes.Buffer)
		_, err = io.Copy(buf, resp.Body)
		if err != nil {
			result.err = err
			SendData(ctx, sender, result)
			return
		}

		result.resp = &Response{
			URL:  url,
			Data: buf.String(),
		}
		SendData(ctx, sender, result)
	}
}
func SendData[T any](ctx context.Context, sender chan<- T, data T) {
	select {
	case <-ctx.Done():
	case sender <- data:
	}
}
