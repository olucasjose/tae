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
   *Para ignorar padrões específicos:*
   ```bash
   tae track frontend/ -i "node_modules|*.tmp" patch_1.2
   ```

3. **Verificar os arquivos que estão na tag:**
   ```bash
   tae list patch_1.2
   ```

4. **Exportar a tag mantendo a hierarquia de pastas (para `./saida`):**
   ```bash
   tae export patch_1.2 ./saida
   ```

## 🛠️ Referência de Comandos

| Comando | Descrição | Exemplo |
|---|---|---|
| `create <tag>` | Cria um novo contexto (tag) vazio no banco de dados. | `tae create refactor_api` |
| `delete <tag>...` | Remove uma ou mais tags e todo o seu índice de rastreamento. | `tae delete tag1 tag2` |
| `list [tag]` | Lista todas as tags cadastradas. Se a tag for informada, lista os caminhos rastreados por ela. | `tae list` |
| `track <alvos> <tag>` | Adiciona arquivos/pastas ao monitoramento da tag. Suporta `-i` (ou `--ignore`) para regex/glob raso. | `tae track ./cmd/ meu_app` |
| `untrack <alvos> <tag>`| Remove arquivos/pastas específicos do monitoramento de uma tag. | `tae untrack ./cmd/main.go meu_app` |
| `export <tag> <dest>` | Exporta os arquivos rastreados. Suporta modo ZIP (`-z`) e limitação de lotes (`-l`). | `tae export meu_app ./build -z -l 500` |
| `diff-zip <c1> <c2>` | Compara dois commits do Git, isola os arquivos modificados/adicionados e empacota em zip no disco. | `tae diff-zip HEAD~1 HEAD -l 100` |

### Detalhes de Exportação e Zip (`export` / `diff-zip`)

Se você trabalhar com milhares de arquivos, os comandos de exportação zipada suportam o fatiamento inteligente de lotes (`--limit` ou `-l`). O algoritmo tenta quebrar os arquivos limitando o total por arquivo `.zip`, separando na raiz dos subdiretórios quando possível.
Para mesclar lotes que fiquem pequenos demais no final do fatiamento, use a flag `--merge` (`-m`).

## 📄 Licença

Distribuído sob a licença Apache 2.0. Veja `LICENSE` para mais informações.
Copyright 2026 Lucas José de Lima Silva.
