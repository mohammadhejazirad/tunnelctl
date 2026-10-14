package domain

import (
	"context"
	"crypto/x509"
	"errors"
	"net"
	"testing"
)

func TestFailureCategoriesDoNotEchoRawErrors(t *testing.T) {
	for _, tc := range []struct {
		e    error
		want string
	}{
		{context.Canceled, "canceled"}, {context.DeadlineExceeded, "timeout"},
		{&net.DNSError{Err: "synthetic-secret", Name: "synthetic-private.invalid"}, "dns"},
		{x509.UnknownAuthorityError{}, "tls"}, {errors.New("synthetic-url-token"), "upstream_or_local_unknown"},
		{NetworkFail("source_policy", "Blocked."), "source_policy"},
	} {
		if got := FailureCategory(tc.e); got != tc.want {
			t.Fatal("unexpected category", got)
		}
	}
}
