package caddy

import (
	"github.com/captncraig/cors"
	"net/http"
	"strconv"
	"strings"

	"github.com/mholt/caddy/caddy/setup"
	"github.com/mholt/caddy/middleware"
)

type corsRule struct {
	Conf *cors.Config
	Path string
}

func Setup(c *setup.Controller) (middleware.Middleware, error) {
	rules, err := parseRules(c)
	if err != nil {
		return nil, err
	}
	return func(next middleware.Handler) middleware.Handler {
		return middleware.HandlerFunc(func(w http.ResponseWriter, r *http.Request) (int, error) {
			for _, rule := range rules {
				if middleware.Path(r.URL.Path).Matches(rule.Path) {
					rule.Conf.HandleRequest(w, r)
					if cors.IsPreflight(r) {
						return 200, nil
					}
					break
				}
			}
			return next.ServeHTTP(w, r)
		})
	}, nil
}

func parseRules(c *setup.Controller) ([]*corsRule, error) {
	rules := []*corsRule{}

	for c.Next() {
		rule := &corsRule{Path: "/", Conf: cors.Default()}
		args := c.RemainingArgs()

		anyOrigins := false
		switch len(args) {
		case 0:
		case 2:
			rule.Conf.AllowedOrigins = strings.Split(c.Val(), ",")
			anyOrigins = true
			fallthrough
		case 1:
			rule.Path = args[0]
		default:
			return nil, c.Errf(`Too many arguments`, c.Val())
		}
		for c.NextBlock() {
			switch c.Val() {
			case "origin":
				if !anyOrigins {
					rule.Conf.AllowedOrigins = []string{}
				}
				args := c.RemainingArgs()
				for _, domain := range args {
					rule.Conf.AllowedOrigins = append(rule.Conf.AllowedOrigins, domain)
				}
				anyOrigins = true
			case "methods":
				if arg, err := singleArg(c, "methods"); err != nil {
					return nil, err
				} else {
					rule.Conf.AllowedMethods = arg
				}
			case "allow_credentials":
				if arg, err := singleArg(c, "allow_credentials"); err != nil {
					return nil, err
				} else {
					var b bool
					if arg == "true" {
						b = true
					} else if arg != "false" {
						return nil, c.Errf("allow_credentials must be true or false.")
					}
					rule.Conf.AllowCredentials = &b
				}
			case "max_age":
				if arg, err := singleArg(c, "max_age"); err != nil {
					return nil, err
				} else {
					i, err := strconv.Atoi(arg)
					if err != nil {
						return nil, c.Err("max_age must be valid int")
					}
					rule.Conf.MaxAge = i
				}
			case "allowed_headers":
				if arg, err := singleArg(c, "allowed_headers"); err != nil {
					return nil, err
				} else {
					rule.Conf.AllowedHeaders = arg
				}
			case "exposed_headers":
				if arg, err := singleArg(c, "exposed_headers"); err != nil {
					return nil, err
				} else {
					rule.Conf.ExposedHeaders = arg
				}
			default:
				return nil, c.Errf("Unknown cors config item: %s", c.Val())
			}
		}
		rules = append(rules, rule)
	}
	return rules, nil
}

func singleArg(c *setup.Controller, desc string) (string, error) {
	args := c.RemainingArgs()
	if len(args) != 1 {
		return "", c.Errf("%s expects exactly one argument", desc)
	}
	return args[0], nil
}