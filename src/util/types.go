package util

type UpstreamSource struct {
	URL    string
	Format string
}

type Rule struct {
	Type  string
	Value string
}

type RewriteRule struct {
	Line     string
	Category string
}