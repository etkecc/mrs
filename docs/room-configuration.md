# Room Configuration

MRS allows you to configure different room parameters by adding special configuration strings to the room topic.

Table of Contents:

<!-- vim-markdown-toc GFM -->

* [How?](#how)
    * [Format](#format)
* [Options](#options)
    * [Language](#language)
    * [Contact Email](#contact-email)
    * [Noindex](#noindex)

<!-- vim-markdown-toc -->

## How?

By adding specially formatted string to the room topic, example:

```
(MRS-language:DE|email:admin@example.com-MRS)
```

The string could be only 1 per room, i.e., multiple `(MRS-...-MRS)` strings are not allowed in the room topic.

The best place to add those options is the end of the room topic, because they will be removed from the topic when MRS will process it

💡 **Hint**: you can "hide" that configuration string by utilizing markdown link format - just use `[.]` (or even `[]`) as the link text, so it will not be visible in the room topic, but still will be processed by MRS. Below is an example of how to do that:

```markdown
[.](MRS-language:EN|email:admin@example.com-MRS)
```

will look like [.](MRS-language:EN|email:admin@example.com-MRS) (_yes, just a dot_) for client apps that support [MSC3765](https://github.com/matrix-org/matrix-spec-proposals/pull/3765).

### Format

* Start tag: `(MRS-`
* Delimiter: `|`
* Values: `key:value`
* End tag: `-MRS)`

* Example: `(MRS-language:EN|email:you@example.com-MRS)`

## Options

### Language

MRS detects the language of a room by using [lingua-go](https://github.com/pemistahl/lingua-go) library against room name and topic.
Unfortunately, the results may be inaccurate, because there is no enough text to detect the language correctly.

To fix that, you, as a room administrator/moderator, can set the language manually by adding the following to the room topic (inside `(MRS-...-MRS)`):

```
language:EN
```

replace `EN` with the language code you want to set, supported languages are: [ISO 639-1](https://en.wikipedia.org/wiki/List_of_ISO_639-1_codes) codes (`Set 1` - 2-letter codes, e.g., `EN`, `RU`, `DE`, `FR`, etc.). Only one language code is allowed per room.

### Contact Email

MRS allows you to add your email address to the room description, so it will be used as a contact email in case of the room is reported,
and it could be used as a proof of ownership (that's up to specific MRS instance) as well. [More details about reports](./msc1929.md)

To do that, you have to add the following to the room topic (inside `(MRS-...-MRS)`):

```
email:admin@example.com
```

### Noindex

MRS allows you to prevent indexing of a specific room by adding the following to the room topic (inside `(MRS-...-MRS)`):

```
noindex:true
```

[More details about deindexing](./deindexing.md)
