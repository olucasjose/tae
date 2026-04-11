# Tae (Tracker and Exporter)

Tae é uma ferramenta de linha de comando (CLI) escrita em Go, desenvolvida para gerenciar, rastrear e extrair arquivos através de um sistema de "tags". Ideal para empacotar patches, exportar alterações de código e fatiar grandes volumes de arquivos em lotes menores.

O sistema opera com um banco de dados local (BoltDB) armazenado em `~/.tae/tae.db`, registrando os caminhos absolutos dos arquivos monitorados.

## 🚀 Instalação

Você deve ter o [Go](https://go.dev/) (1.25+) instalado. Clone/extraia o repositório e execute o script de instalação para compilar e mover o binário para o seu `PATH`.

```bash
chmod +x install.sh
./install.sh
```

O script detecta automaticamente se você está em um ambiente Linux padrão ou no Termux (Android) e fará o roteamento adequado.

## 💡 Como Funciona (Guia Rápido)

O fluxo principal baseia-se em: **Criar uma Tag** -> **Rastrear Arquivos** -> **Exportar**.

1. **Criar a tag:**
   ```bash
   tae create patch_1.2
   ```

2. **Rastrear arquivos ou diretórios inteiros:**
   *(Nota: O nome da tag sempre vai no final do comando)*
   ```bash
   tae track src/handlers/ api/routes.go patch_1.2
   ```
   *Se quiser que um arquivo interno de uma pasta rastreada NÂO seja exportado, coloque-o na blacklist permanente da tag:*
   ```bash
   tae ignore src/handlers/dev_test.go patch_1.2
   ```

3. **Verificar os arquivos que estão na tag:**
   ```bash
   tae list patch_1.2
   ```
   *Para auditar a blacklist:*
   ```bash
   tae list patch_1.2 --ignored
   ```

4. **Exportar a tag mantendo a hierarquia de pastas (para `./saida`):**
   ```bash
   tae export patch_1.2 ./saida
   ```

*(Nota: Para verificar a versão atual da CLI, utilize `tae -v` ou `tae --version`)*

## 🛠️ Referência de Comandos

| Comando | Descrição | Exemplo |
|---|---|---|
| `create <tags>...` | Cria novos contextos (tags) vazios no banco de dados. | `tae create refactor fix` |
| `delete <tags>...` | Remove uma ou mais tags e todo o seu índice de rastreamento. | `tae delete tag1 tag2` |
| `list [tag]` | Lista alvos rastreados. Suporta árvore visual (`-t`), profundidade (`-L`), filtro (`-I`), expansão de pastas (`-e`) e dump de exclusões (`-i` / `--ignored`). | `tae list refactor -t -e` |
| `track <alvos> <tag>` | Adiciona arquivos/pastas ao monitoramento explícito da tag. | `tae track ./cmd/ meu_app` |
| `untrack <alvos> <tag>`| Remove arquivos/pastas específicos do monitoramento explícito. | `tae untrack ./cmd/ meu_app` |
| `ignore <alvos> <tag>` | Adiciona arquivos à blacklist (Exclusion Index) para silenciá-los. Use `-r` para remover da blacklist. | `tae ignore ./src/tmp meu_app` |
| `prune [tags]...` | Remove arquivos fantasmas do banco. Exige confirmação por padrão. Suporta listar (`-l`), forçar (`-f`), silencioso (`-q`) e todas as tags (`-a`). | `tae prune -a -f` |
| `export <tag> <dest>` | Exporta os arquivos rastreados lendo o disco local. Suporta fatiamento em zip (`-z`, `-l`, `-m`). | `tae export meu_app ./build -z` |
| `git diff <c1> <c2>` | Empacota em zip os arquivos alterados (isolado da working tree). O nome do zip detecta automaticamente o repositório. | `tae git diff HEAD~1 HEAD -l 100` |
| `git list <commit>` | Lista os arquivos da árvore de um commit. Suporta as mesmas opções visuais do `list` (`-t`, `-L`, `-I`). | `tae git list HEAD -t -L 1` |
| `git export <c> <dest>`| Exporta a árvore de um commit. Identifica o repositório raiz e gera lotes nomeados de forma inteligente. | `tae git export v3.2.0 ./saida -z` |

### Detalhes de Exportação e Zip (`export` / `git diff` / `git export`)

Se você trabalhar com milhares de arquivos, os comandos de exportação zipada suportam o fatiamento inteligente de lotes (`--limit` ou `-l`). O algoritmo tenta quebrar os arquivos limitando o total por arquivo `.zip`, separando na raiz dos subdiretórios quando possível.
Para mesclar lotes que fiquem pequenos demais no final do fatiamento, use a flag `--merge` (`-m`).

Nas integrações do Git (`git export` e `git diff`), o tae extrai automaticamente o nome do repositório raiz atual para nomear os zips gerados, garantindo rastreabilidade dos pacotes.

## 📄 Licença

Distribuído sob a licença Apache 2.0. Veja `LICENSE` para mais informações.
Copyright 2026 Lucas José de Lima Silva.