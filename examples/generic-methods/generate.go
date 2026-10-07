package arith

// arith.wasm and arith_visitor/ are both derived from arith.ohm and drift
// apart silently if only one is regenerated, so rerun both after editing the
// grammar. The docker tag must be one of the versions the runtime accepts,
// see ohm.(*Grammar).MatchingDockerImageTags.
//
//go:generate docker run --rm -v "$PWD:/local" ohmjs/ohm:18.0.0-beta.15 compile arith.ohm
//go:generate go run github.com/ohmjs/ohm-go/ohm-cli generate go --generic-methods -P arith_visitor --output-dir arith_visitor @arith.ohm
