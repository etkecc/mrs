# Room Configuration

MRS allows you to configure different room parameters by adding special configuration strings to the room topic/description.

Table of Contents:

<!-- vim-markdown-toc GFM -->

* [How?](#how)
* [Options](#options)
    * [Language](#language)
    * [Contact Email](#contact-email)

<!-- vim-markdown-toc -->

## How?

All config options follow the same format, use it in the room description:

```
<matrix.server_name from MRS config.yml>:<option>:<value>
```

Example:

```
example.com:email:admin@example.com
```

The best place to add those options is the end of the room topic, because they will be removed from the topic when MRS will process it
(that is purely technical information and workaround, so it should not be indexed and it not meant to be searchable via MRS).

All examples below will use the [MatrixRooms.info](https://matrixrooms.info) - the demo server for MRS.

## Options

### Language

MRS detects the language of a room by using [lingua-go](https://github.com/pemistahl/lingua-go) library against room name and topic.
Unfortunately, the results may be inaccurate, because there is no enough text to detect the language correctly.

To fix that, you, as a room administrator/moderator, can set the language manually by adding the following to the room description:

```
matrixrooms.info:language:EN
```

replace `EN` with the language code you want to set, supported languages are: [ISO 639-1](https://en.wikipedia.org/wiki/List_of_ISO_639-1_codes) codes (`Set 1` - 2-letter codes, e.g., `EN`, `RU`, `DE`, `FR`, etc.).

### Contact Email

MRS allows you to add your email address to the room description, so it will be used as a contact email in case of the room is reported,
and it could be used as a proof of ownership (that's up to specific MRS instance) as well. [More details about reports](./msc1929.md)

To do that, you have to add the following to the room description:

```
matrixrooms.info:email:your@email.address
```
