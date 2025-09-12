//go:build mcp

package mcpclient

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"os/exec"
	"strings"

	jsonschema "github.com/google/jsonschema-go/jsonschema"
	mcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Client defines minimal MCP client capabilities we need.
type Client interface {
	Handshake(ctx context.Context) error
	ListTools(ctx context.Context) ([]ToolDescriptor, error)
	CallTool(ctx context.Context, name string, args map[string]any) (map[string]any, error)
	ListResources(ctx context.Context) ([]ResourceDescriptor, error)
	Close() error
}

type Option func(*config)

type config struct{}

// ToolDescriptor is a subset of MCP tool schema.
type ToolDescriptor struct {
	Name         string
	Description  string
	InputSchema  []byte
	OutputSchema []byte
}

// ResourceDescriptor describes an MCP resource.
type ResourceDescriptor struct {
	URI         string
	Description string
}

type sdkClient struct {
	client  *mcp.Client
	session *mcp.ClientSession
}

// New creates an MCP client and connects immediately using the v0.4 SDK.
// Address schemes:
//   - cmd:<program> [<args...>]  (e.g., cmd:./server)
//   - http(s)://...              (SSE client transport)
func New(ctx context.Context, addr string, _ ...Option) (Client, error) {
	if strings.HasPrefix(addr, "cmd:") {
		prog := strings.TrimPrefix(addr, "cmd:")
		fields := strings.Fields(prog)
		if len(fields) == 0 {
			return nil, errors.New("cmd: missing program")
		}
		cmd := exec.Command(fields[0], fields[1:]...)
		transport := &mcp.CommandTransport{Command: cmd}
		c := mcp.NewClient(&mcp.Implementation{Name: "orch-mcp-client", Version: "dev"}, nil)
		sess, err := c.Connect(ctx, transport, nil)
		if err != nil {
			return nil, err
		}
		return &sdkClient{client: c, session: sess}, nil
	}
	// Default: treat as URL; support http/https for SSE
	u, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}
	switch u.Scheme {
	case "http", "https":
		transport := &mcp.SSEClientTransport{Endpoint: u.String()}
		c := mcp.NewClient(&mcp.Implementation{Name: "orch-mcp-client", Version: "dev"}, nil)
		sess, err := c.Connect(ctx, transport, nil)
		if err != nil {
			return nil, err
		}
		return &sdkClient{client: c, session: sess}, nil
	default:
		return nil, errors.New("unsupported MCP client scheme: " + u.Scheme)
	}
}

func (s *sdkClient) Handshake(context.Context) error {
	// v0.4 establishes session on Connect; no explicit handshake needed.
	if s.session == nil {
		return errors.New("no active session")
	}
	return nil
}

func (s *sdkClient) ListTools(ctx context.Context) ([]ToolDescriptor, error) {
	if s.session == nil {
		return nil, errors.New("no active session")
	}
	res, err := s.session.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		return nil, err
	}
	out := make([]ToolDescriptor, 0, len(res.Tools))
	for _, t := range res.Tools {
		var inBytes, outBytes []byte
		if t.InputSchema != nil {
			inBytes, _ = json.Marshal(t.InputSchema)
		}
		// Output schema may not be present in listing; keep empty to avoid implying stricter contract
		if ts, ok := any(t).(interface{ OutputSchema() *jsonschema.Schema }); ok {
			if sch := ts.OutputSchema(); sch != nil {
				outBytes, _ = json.Marshal(sch)
			}
		}
		out = append(out, ToolDescriptor{Name: t.Name, Description: t.Description, InputSchema: inBytes, OutputSchema: outBytes})
	}
	return out, nil
}

func (s *sdkClient) CallTool(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	if s.session == nil {
		return nil, errors.New("no active session")
	}
	res, err := s.session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		return nil, err
	}
	if res.IsError {
		return nil, errors.New("tool returned error")
	}
	m, _ := res.StructuredContent.(map[string]any)
	return m, nil
}

func (s *sdkClient) ListResources(ctx context.Context) ([]ResourceDescriptor, error) {
	if s.session == nil {
		return nil, errors.New("no active session")
	}
	res, err := s.session.ListResources(ctx, &mcp.ListResourcesParams{})
	if err != nil {
		return nil, err
	}
	out := make([]ResourceDescriptor, 0, len(res.Resources))
	for _, r := range res.Resources {
		out = append(out, ResourceDescriptor{URI: r.URI, Description: r.Description})
	}
	return out, nil
}

func (s *sdkClient) Close() error {
	if s.session != nil {
		_ = s.session.Close()
	}
	return nil
}
