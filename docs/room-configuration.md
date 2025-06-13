# Room Configuration

MRS allows you to configure different room parameters by adding special configuration strings to the room's topic.

Table of Contents:

<!-- vim-markdown-toc GFM -->

* [How?](#how)
    * [Format](#format)
* [Options](#options)
    * [Language](#language)
    * [Contact Email address](#contact-email-address)
    * [Noindex](#noindex)

<!-- vim-markdown-toc -->

## How?

You can configure different room parameters by adding specially formatted string to the room's topic, following the example below:

```
(MRS-language:DE|email:admin@example.com-MRS)
```

The string can be only 1 per room. If multiple `(MRS-...-MRS)` strings are set, they will not be properly recognized.

The best place to add those parameters is the end of the room's topic, as they will be removed from the topic when MRS will process it.

💡 **Hint**: you can "hide" that configuration string by utilizing Markdown link format — just use `[.]` (or even `[]`) as the link text, so it will not be visible in the room's topic, but still will be processed by MRS. Below is an example of how to do that:

```markdown
[.](MRS-language:EN|email:admin@example.com-MRS)
```

This will make the string displayed as [.](MRS-language:EN|email:admin@example.com-MRS) (_yes, just a dot_) on client apps that support [MSC3765](https://github.com/matrix-org/matrix-spec-proposals/pull/3765).

### Format

* Start tag: `(MRS-`
* Delimiter: `|`
* Values: `key:value`
* End tag: `-MRS)`

* Example: `(MRS-language:EN|email:you@example.com-MRS)`

## Options

### Language

MRS detects the language of a room based on the room's name and topic by using [lingua-go](https://github.com/pemistahl/lingua-go) library. The result may be inaccurate, due to the length of text to detect the language correctly.

If you are a room administrator or moderator, you can manually specify the language on the room's topic (inside `(MRS-...-MRS)`):

```
language:EN
```

Please replace `EN` with the language code, following the [ISO 639-1](https://en.wikipedia.org/wiki/List_of_ISO_639-1_codes) format (`Set 1` - 2-letter codes, e.g., `EN`, `RU`, `DE`, `FR`, etc.). Only one language code is allowed per room.

### Contact Email address

MRS allows you to add your email address to the room description, so it will be used as a contact email in case of the room is reported, and it could be used as a proof of ownership (that's up to specific MRS instance) as well. You can check [this page](./msc1929.md) for details about reports.

To specify the contact Email address, add the following to the room's topic (inside `(MRS-...-MRS)`):

```
email:admin@example.com
```

### Noindex

To prevent MRS instances from indexing a specific room, add the following to the room's topic (inside `(MRS-...-MRS)`):

```
noindex:true
```

Refer to [this page](./deindexing.md) for more details about deindexing.
