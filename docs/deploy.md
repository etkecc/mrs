<!--
SPDX-FileCopyrightText: 2023 Nikita Chernyi
SPDX-FileCopyrightText: 2025 Suguru Hirahara

SPDX-License-Identifier: AGPL-3.0-or-later
-->

# Deploy

<!-- vim-markdown-toc GitLab -->

* [Manual](#manual)
* [Ansible](#ansible)

<!-- vim-markdown-toc -->

## Manual

1. Build Matrix Rooms Search
2. Copy `config.yml.sample` into `config.yml` and adjust it
3. Run `mrs -genkey` to add the key to the config
4. Run Matrix Rooms Search with `-c config.yml`
5. You probably want to call `/-/full` admin API endpoint at start

## Ansible

Matrix Rooms Search is integrated with the [MASH playbook](https://github.com/mother-of-all-self-hosting/mash-playbook/). Check [its documentation](https://github.com/mother-of-all-self-hosting/mash-playbook/blob/main/docs/services/mrs.md) for details.
