<!--
SPDX-FileCopyrightText: 2025 Nikita Chernyi
SPDX-FileCopyrightText: 2025 Suguru Hirahara

SPDX-License-Identifier: AGPL-3.0-or-later
-->

# Room Configuration

Table of Contents:

<!-- vim-markdown-toc GFM -->

- [Synopsis](#synopsis)
- [Format](#format)
- [Options](#options)
    - [Language](#language)
    - [Contact Email address](#contact-email-address)
    - [Noindex](#noindex)

<!-- vim-markdown-toc -->

## Synopsis

You can direct Matrix Rooms Search instances to recognize rooms per your settings by adding a specially formatted string to the room's topic, following the example below:

```
(MRS-language:DE|email:admin@example.com-MRS)
```

The recommended place to add the string is the end of the room's topic, as it will be removed from the topic when MRS will process it.

💡 **Hint**: you can "hide" that configuration string by utilizing Markdown link format — just use `[.]` (or even `[]`) as the link text, so it will not be visible in the room's topic, but still will be processed by MRS. Below is an example of how to do that:

```markdown
[.](MRS-language:EN|email:admin@example.com-MRS)
```

This will render the string as [.](MRS-language:EN|email:admin@example.com-MRS) (_yes, just a dot_) on client apps that support [MSC3765](https://github.com/matrix-org/matrix-spec-proposals/pull/3765).

## Format

- Start tag: `(MRS-`
- Delimiter: `|`
- Values: `key:value`
- End tag: `-MRS)`

Here is an example of a string to add settings about the room's language to _English_ and the contact email address to _<you@example.com>_: `(MRS-language:EN|email:you@example.com-MRS)`

Only one string is allowed per room. If multiple `(MRS-...-MRS)` strings are set, they will not be properly recognized.

## Options

### Language

MRS detects the language of a room based on the room's name and topic by using [lingua-go](https://github.com/pemistahl/lingua-go) library. The result may be inaccurate, due to the length of text to detect the language correctly.

If your permission level enables you to edit the room's topic, you can manually specify the language on the room's topic with the tag. For example, if it is to be specified with French, add the value to the tag as below:

```
language:FR
```

[ISO 639-1](https://en.wikipedia.org/wiki/List_of_ISO_639-1_codes) format (`Set 1` - 2-letter codes, e.g., `EN`, `RU`, `DE`, `FR`, etc.) is recognized as a value. Note that only one language code is accepted per room.

### Contact Email address

You can add your email address to the string for receiving a notification from a MRS instance in case the room is reported. It can also be used as a proof of ownership (that's up to specific MRS instance). You can check [this page](./msc1929.md) for details about reports.

To specify the contact Email address, add the following value to the tag:

```
email:admin@example.com
```

### Noindex

To prevent MRS instances from indexing a specific room, specify the value below to the tag (inside `(MRS-...-MRS)`):

```
noindex:true
```

Refer to [this page](./deindexing.md) for more details about deindexing.
