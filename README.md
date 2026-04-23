# Tae (Tracker and Exporter)

Tae é uma ferramenta de linha de comando (CLI) escrita em Go, desenvolvida para gerenciar, rastrear e extrair arquivos através de um sistema de "tags". Ideal para empacotar patches, exportar alterações de código, fazer backup de metadados do Git e fatiar grandes volumes de arquivos em lotes menores.

O sistema opera com um banco de dados local (BoltDB) armazenado em `~/.tae/tae.db`, registrando os caminhos absolutos dos arquivos monitorados.

**Versão atual:** 6.7.1

## 🚀 Instalação

Você deve ter o [Go](https://go.dev/) (1.25+) instalado. Clone/extraia o repositório e execute o script de instalação para compilar e mover o binário para o seu `PATH`.

```bash
chmod +x install.sh
./install.sh
```

O script detecta automaticamente se você está em um ambiente Linux padrão ou no Termux (Android) e fará o roteamento adequado.

### Autocompletar

Para habilitar o autocompletar com [TAB] no terminal, gere o script correspondente ao seu shell:

**Linux (Bash):**
```bash
tae completion bash | sudo tee /etc/bash_completion.d/tae > /dev/null
exec bash
```

**Termux (Android):**
```bash
tae completion bash > /data/data/com.termux/files/usr/etc/bash_completion.d/tae
exit  # Reinicie a sessão do terminal completamente
```

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
   *Se quiser que um arquivo interno de uma pasta rastreada NÃO seja exportado, coloque-o na denylist permanente da tag:*
   ```bash
   tae ignore src/handlers/dev_test.go patch_1.2
   ```

3. **Verificar os arquivos que estão na tag:**
   ```bash
   tae list patch_1.2
   ```
   *Para auditar a denylist:*
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
| `rename <old> <new>`| Renomeia uma tag existente e migra seus metadados. | `tae rename fix bugfix` |
| `delete <tags>...` | Remove uma ou mais tags e todo o seu índice de rastreamento. | `tae delete tag1 tag2` |
| `convert <tag>` | Converte uma tag entre escopos Local e Git. Use `-g` para Git ou `-t` para Local. | `tae convert minha_tag -g` |
| `list [tag]` | Lista alvos rastreados. Suporta árvore visual (`-t`), profundidade (`-L`), filtro (`-I`), expansão de pastas (`-e`) e dump de exclusões (`-i` / `--ignored`). | `tae list refactor -t -e` |
| `track <alvos> <tag>` | Adiciona arquivos/pastas ao monitoramento explícito da tag. | `tae track ./cmd/ meu_app` |
| `untrack <alvos> <tag>`| Remove arquivos/pastas específicos do monitoramento explícito. | `tae untrack ./cmd/ meu_app` |
| `ignore <alvos> <tag>` | Adiciona arquivos à denylist (Exclusion Index) para silenciá-los. Use `-r` para remover da denylist. | `tae ignore ./src/tmp meu_app` |
| `prune [tags]...` | Remove arquivos fantasmas do banco. Exige confirmação por padrão. Suporta listar (`-l`), forçar (`-f`), silencioso (`-q`) e todas as tags (`-a`). | `tae prune -a -f` |
| `export <tag> <dest>` | Exporta os arquivos rastreados lendo o disco local. Suporta fatiamento em zip (`-z`, `-l`, `-m`). | `tae export meu_app ./build -z` |
| `git diff <c1> <c2>` | Empacota em zip os arquivos alterados (isolado da working tree). O nome do zip detecta automaticamente o repositório. | `tae git diff HEAD~1 HEAD -l 100` |
| `git list <commit>` | Lista os arquivos da árvore de um commit. Suporta as mesmas opções visuais do `list` (`-t`, `-L`, `-I`). | `tae git list HEAD -t -L 1` |
| `git export <c> <dest>`| Exporta a árvore de um commit. Identifica o repositório raiz e gera lotes nomeados de forma inteligente. | `tae git export v3.2.0 ./saida -z` |
| `git backup-save [dir]` | Exporta tags e denylists do repositório Git para um arquivo JSON. Use `-a`, `-d`, `-t` ou `-o` para definir o escopo. | `tae git backup-save ./backups -a` |
| `git backup-restore <file>` | Importa tags e denylists de um arquivo de backup JSON para o repositório Git atual. | `tae git backup-restore repo_20260102_tae-backup.json` |
| `completion <shell>` | Gera script de autocompletar para bash, zsh, fish ou powershell. | `tae completion bash` |

### Detalhes de Exportação e Zip (`export` / `git diff` / `git export`)

Se você trabalhar com milhares de arquivos, os comandos de exportação zipada suportam o fatiamento inteligente de lotes (`--limit` ou `-l`). O algoritmo tenta quebrar os arquivos limitando o total por arquivo `.zip`, separando na raiz dos subdiretórios quando possível.
Para mesclar lotes que fiquem pequenos demais no final do fatiamento, use a flag `--merge` (`-m`).

Nas integrações do Git (`git export` e `git diff`), o tae extrai automaticamente o nome do repositório raiz atual para nomear os zips gerados, garantindo rastreabilidade dos pacotes.

### Backup e Restore no Git (`git backup-save` / `git backup-restore`)

Os comandos de backup permitem exportar e importar metadados de tags associadas a um repositório Git específico. O arquivo JSON gerado contém:
- **RepoID**: Identificador único do repositório (hash do caminho)
- **RepoName**: Nome do repositório
- **RepoDenylist**: Lista de exclusão global do repositório
- **Tags**: Conjunto de tags com seus arquivos e denylists associadas

**Flags do `git backup-save`:**
- `-a, --all`: Exporta tudo (denylist do repo e todas as tags)
- `-d, --denylist`: Exporta apenas a denylist do repositório
- `-t, --tag`: Exporta todas as tags do git e suas denylists
- `-o, --only <tags>`: Exporta apenas as tags listadas (ex: `-o tag1,tag2,denylist`)

**Importante:** O restore só funciona se o RepoID do backup corresponder ao repositório atual, garantindo que os metadados sejam restaurados apenas no repositório original.

### Conversão de Escopo (`convert`)

O comando `convert` permite migrar uma tag entre os escopos **Local** (padrão, baseado em caminhos absolutos) e **Git** (baseado em caminhos relativos ao repositório).

- **Local → Git (`-g`):** Requer estar dentro de um repositório Git. Caminhos absolutos são convertidos para relativos.
- **Git → Local (`-t`):** Requer estar dentro do repositório Git original. Caminhos relativos são expandidos para absolutos.

## 📄 Licença

Distribuído sob a licença Apache 2.0. Veja `LICENSE` para mais informações.
Copyright 2026 Lucas José de Lima Silva.