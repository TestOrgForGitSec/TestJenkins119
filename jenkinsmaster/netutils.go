package jenkinsmaster

import (
	"github.com/deliveryblueprints/chlog-go/log"
	"net/http"
	"net/http/httputil"
)

type loggingTransport struct{}

func (s *loggingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	bytes, _ := httputil.DumpRequestOut(r, true)
	log.Debug().Msgf("Jenkins request is: %s", bytes)

	values := r.URL.Query()
	values.Add("tree", "jobs[name,url],*") //AdditionalA parameter added
	r.URL.RawQuery = values.Encode()

	log.Debug().Msgf("Jenkins request URL: %s", r.URL.String())
	resp, err := http.DefaultTransport.RoundTrip(r)

	respBytes, _ := httputil.DumpResponse(resp, true)
	bytes = append(bytes, respBytes...)
	log.Debug().Msgf("Jenkins response is: %s", bytes)
	return resp, err
}

func GetHttpClient() http.Client {
	client := http.Client{
		Transport: &loggingTransport{},
	}
	return client
}
