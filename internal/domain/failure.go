package domain

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
)

// NetworkFailure retains only an evidence-based category, never the raw URL error.
type NetworkFailure struct {
	Public   *Error
	Category string
}

func (e *NetworkFailure) Error() string { return e.Public.Message }
func (e *NetworkFailure) Unwrap() error { return e.Public }
func NetworkFail(category, message string) error {
	return &NetworkFailure{&Error{5, message, "Check DNS, TLS trust, endpoint availability and source policy; no insecure fallback is used."}, category}
}
func FailureCategory(e error) string {
	if e == nil {
		return "none"
	}
	var f *NetworkFailure
	if errors.As(e, &f) {
		return f.Category
	}
	if errors.Is(e, context.Canceled) {
		return "canceled"
	}
	if errors.Is(e, context.DeadlineExceeded) {
		return "timeout"
	}
	var dns *net.DNSError
	if errors.As(e, &dns) {
		if dns.IsTimeout {
			return "timeout"
		}
		return "dns"
	}
	var verify *tls.CertificateVerificationError
	var unknown x509.UnknownAuthorityError
	var invalid x509.CertificateInvalidError
	var hostname x509.HostnameError
	if errors.As(e, &verify) || errors.As(e, &unknown) || errors.As(e, &invalid) || errors.As(e, &hostname) {
		return "tls"
	}
	var ne net.Error
	if errors.As(e, &ne) && ne.Timeout() {
		return "timeout"
	}
	var de *Error
	if errors.As(e, &de) {
		switch de.Code {
		case 2:
			return "invalid_input"
		case 3:
			return "unsupported"
		case 4:
			return "local_permission"
		case 6:
			return "local_conflict"
		case 7:
			return "cleanup"
		}
	}
	return "upstream_or_local_unknown"
}
