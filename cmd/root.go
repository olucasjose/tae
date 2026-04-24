// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package cmd

import (
	"fmt"
	"os"

	"tae/internal/storage"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:           "tae",
	Version:       "7.0.0",
	Short:         "Tae é um utilitário CLI para extração e empacotamento de código",
	SilenceErrors: true,
	SilenceUsage:  true,
	Long: `Tae (Tracker and Exporter) é uma ferramenta CLI modular para gerenciar, 
rastrear e extrair arquivos no disco.

O tae utiliza um banco de dados local para criar "tags" (contextos) e associar 
caminhos de arquivos e diretórios a essas tags. Isso permite agrupar arquivos 
espalhados pelo sistema e exportá-los ou compactá-los de uma vez só, mantendo 
a estrutura de diretórios original e calculando os agrupamentos.

Dicas de Autocompletar:
Para habilitar o [TAB] no terminal, gere o script correspondente ao seu shell.
Exemplo Linux (Bash):
  tae completion bash | sudo tee /etc/bash_completion.d/tae > /dev/null
  # ou
  sudo sh -c 'tae completion bash > /etc/bash_completion.d/tae'
  exec bash

Exemplo Termux (Android):
  tae completion bash > /data/data/com.termux/files/usr/etc/bash_completion.d/tae
  exit (Reinicie a sessão do terminal completamente)`,
	Example: `  # Criar tags e rastrear arquivos
  tae create "patch_v1" "patch_v2"
  tae track ./src/main.go ./configs/ "patch_v1"
  tae rename "patch_v1" "patch_v2"

  # Gerenciar Denylist Permanente (Exclusion Index)
  tae ignore ./configs/secrets.json "patch_v1"
  tae ignore -r ./configs/secrets.json "patch_v1"
  tae list "patch_v1" --ignored

  # Manutenção e Limpeza (Arquivos deletados do disco)
  tae prune "patch_v1"
  tae prune --all --list
  tae prune -a -f -q

  # Exportações de Tag
  tae export "patch_v1" ./pasta_de_saida
  tae export "patch_v1" ./pasta_de_saida --zip --limit 1000

  # Integrações Git
  tae git list HEAD
  tae git export HEAD~1 ./exportacao_commit -z -l 500`,
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	rootCmd.SetUsageTemplate(`Uso:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [comando]{{end}}{{if .HasExample}}

Exemplos:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

Comandos Disponíveis:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{if eq .Name "help"}}Exibe informações de ajuda para os comandos{{else}}{{.Short}}{{end}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Flags Globais:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [comando] --help" para mais informações sobre um comando específico.{{end}}
`)
}

func Execute() {
	// Garante o fechamento do banco e sincronização do WAL antes de encerrar
	err := rootCmd.Execute()
	if dbErr := storage.CloseDB(); dbErr != nil {
		fmt.Fprintf(os.Stderr, "Aviso: Falha ao fechar o banco de dados: %v\n", dbErr)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
		os.Exit(1)
	}
}
