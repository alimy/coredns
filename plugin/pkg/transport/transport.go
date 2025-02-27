package transport

// These transports are supported by CoreDNS.
const (
	DNS   = "dns"
	TLS   = "tls"
	GRPC  = "grpc"
	HTTPS = "https"
	HTTP = "http"
)

// Port numbers for the various transports.
const (
	// Port is the default port for DNS
	Port = "53"
	// TLSPort is the default port for DNS-over-TLS.
	TLSPort = "853"
	// GRPCPort is the default port for DNS-over-gRPC.
	GRPCPort = "443"
	// HTTPSPort is the default port for DNS-over-HTTPS.
	HTTPSPort = "443"
	// HTTPPort is the default port for DNS-over-HTTP.
	HTTPPort = "80"
)
