# 🌶 exec-sanitize

pass-through execute a command while sanitizing its output. regex or plaintext patterns can be replaced with pre-defined strings. an optional logging dir may be used to store the original values.

```
$ echo "Hi, welcome to Chili's. Bye."

Hi, welcome to Chili's. Bye.
```

```
$ exec-sanitize \
    -p:regex '(Hi|Bye)' \
    -r '<greeting-*>' \
    -p:plain '.*welcome to' \
    -r 'you have arrived at' \
    -log /tmp/log \
    -- \
    bash -c "echo \"Hi, .*welcome to Chili's. Bye.\""

<greeting-0>, you have arrived at Chili's. <greeting-1>.
```

with `/tmp/log` containing the files:

* `0`: `Hi`
* `1`: `Bye`
* `2`: `.*welcome to`

---

```
usage: exec-sanitize <patterns and replacements> -- <command> [args...]

each pattern must be directly followed with replacement. a replacement value of "@discard" deletes the line entirely.

        -log value
                optional directory to log substituted strings as numbered files. if set, replacements will have the first asterisk * replaced with the log item number
        -p:regex value
                regexp pattern to sanitize.
        -p:plain value
                plaintext pattern to sanitize.
        -r value
                what to replace matched substrings with.
```
