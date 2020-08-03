# 🌶 exec-sanitize

pass-through execute a command while sanitizing its output. regex or plaintext patterns can be replaced with pre-defined strings. 

```
$ echo "Hi, welcome to Chili's. Bye."

Hi, welcome to Chili's. Bye.
```

```
$ exec-sanitize \
    -pattern '(Hi|Bye)' \
    -replacement 'Greetings' \
    -plain-pattern 'welcome to' \
    -replacement 'you have arrived at' \
    bash -c "echo \"Hi, welcome to Chili's. Bye.\""

Greetings, you have arrived at Chili's. Greetings.
```

---

```
Usage of exec-sanitize:
  -pattern value
        regexp pattern to sanitize. can be set multiple times
  -plain-pattern value
        plaintext pattern to sanitize. can be set multiple times
  -replacement value
        what to replace matched substrings with. if unset, matches are deleted. if set once, all matches are replaced with the set replacement. if set more than once, there must be a replacement corresponding to each provided pattern (regexp first, then plaintext)
```
