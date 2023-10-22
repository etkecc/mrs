# Deploy

<!-- vim-markdown-toc GitLab -->

* [Manual](#manual)
* [Ansible](#ansible)

<!-- vim-markdown-toc -->

## Manual

1. Build mrs
2. Copy `config.yml.sample` into `config.yml` and adjust it
3. Run mrs with `-c config.yml`
4. You probably want to call `/-/full` admin API endpoint at start

## Ansible

MRS is fully integrated into the [MASH Playbook](https://github.com/mother-of-all-self-hosting/mash-playbook/),
just use the [playbook docs](https://github.com/mother-of-all-self-hosting/mash-playbook/blob/main/docs/services/mrs.md).
