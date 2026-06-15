package urbandictionary

import (
	"context"

	"github.com/tamnd/any-cli/kit"
)

func init() { kit.Register(Domain{}) }

// Domain is the urbandictionary driver. It carries no state; the per-run client is
// built by the factory Register hands kit.
type Domain struct{}

// Info describes the scheme, the hostnames a pasted link is matched against, and
// the identity reused for the binary's help and version.
func (Domain) Info() kit.DomainInfo {
	return kit.DomainInfo{
		Scheme: "urbandictionary",
		Hosts:  []string{Host},
		Identity: kit.Identity{
			Binary: "urbandictionary",
			Short:  "A command line for Urban Dictionary.",
			Long: `A command line for Urban Dictionary.

urbandictionary reads public Urban Dictionary data over plain HTTPS, shapes it into
clean records, and prints output that pipes into the rest of your tools. No API
key, nothing to run alongside it.`,
			Site: Host,
			Repo: "https://github.com/tamnd/urbandictionary-cli",
		},
	}
}

// Register installs the client factory and every operation onto app.
func (Domain) Register(app *kit.App) {
	app.SetClient(newClient)

	kit.Handle(app, kit.OpMeta{Name: "define", Group: "read", List: true,
		Summary: "Look up a slang term on Urban Dictionary",
		Args:    []kit.Arg{{Name: "term", Help: "slang term to define"}}}, defineWord)

	kit.Handle(app, kit.OpMeta{Name: "random", Group: "read", List: true,
		Summary: "Get random Urban Dictionary definitions"}, randomDefinitions)

	kit.Handle(app, kit.OpMeta{Name: "autocomplete", Group: "read", List: true,
		Summary: "Get autocomplete suggestions for a prefix",
		Args:    []kit.Arg{{Name: "prefix", Help: "prefix to autocomplete"}}}, autocomplete)
}

// newClient builds the client from the host-resolved config.
func newClient(_ context.Context, cfg kit.Config) (any, error) {
	c := NewClient()
	if cfg.UserAgent != "" {
		c.UserAgent = cfg.UserAgent
	}
	if cfg.Rate > 0 {
		c.Rate = cfg.Rate
	}
	if cfg.Retries > 0 {
		c.Retries = cfg.Retries
	}
	if cfg.Timeout > 0 {
		c.HTTP.Timeout = cfg.Timeout
	}
	return c, nil
}

// --- inputs ---

type defineInput struct {
	Term   string  `kit:"arg" help:"slang term to define"`
	Limit  int     `kit:"flag,inherit" help:"max results"`
	Client *Client `kit:"inject"`
}

type randomInput struct {
	Limit  int     `kit:"flag,inherit" help:"max results"`
	Client *Client `kit:"inject"`
}

type autocompleteInput struct {
	Prefix string  `kit:"arg" help:"prefix to autocomplete"`
	Client *Client `kit:"inject"`
}

// --- handlers ---

func defineWord(ctx context.Context, in defineInput, emit func(*Definition) error) error {
	defs, err := in.Client.DefineWord(ctx, in.Term)
	if err != nil {
		return err
	}
	for i, d := range defs {
		if in.Limit > 0 && i >= in.Limit {
			break
		}
		if err := emit(d); err != nil {
			return err
		}
	}
	return nil
}

func randomDefinitions(ctx context.Context, in randomInput, emit func(*Definition) error) error {
	defs, err := in.Client.RandomDefinitions(ctx)
	if err != nil {
		return err
	}
	for i, d := range defs {
		if in.Limit > 0 && i >= in.Limit {
			break
		}
		if err := emit(d); err != nil {
			return err
		}
	}
	return nil
}

func autocomplete(ctx context.Context, in autocompleteInput, emit func(*Suggestion) error) error {
	suggestions, err := in.Client.Autocomplete(ctx, in.Prefix)
	if err != nil {
		return err
	}
	for _, s := range suggestions {
		if err := emit(s); err != nil {
			return err
		}
	}
	return nil
}
