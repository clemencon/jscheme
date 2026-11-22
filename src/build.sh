#!/bin/sh
# Builds bin/jscheme from src/: compiles the Scheme program and writes a
# launcher that runs it with the absolute path of the Chez Scheme binary
# (the tests run the interpreter with an empty PATH).
set -eu
root=$(cd "$(dirname "$0")/.." && pwd)
scheme=$(command -v scheme)
mkdir -p "$root/bin"
cd "$root/src"
"$scheme" -q <<EOF
(parameterize ([compile-imported-libraries #t] [generate-inspector-information #f])
  (compile-program "main.ss" "$root/bin/jscheme.so")
  (void))
EOF
cat > "$root/bin/jscheme" <<EOF
#!/bin/sh
exec "$scheme" --program "$root/bin/jscheme.so" "\$@"
EOF
chmod +x "$root/bin/jscheme"
