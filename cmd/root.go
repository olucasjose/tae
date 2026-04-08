// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "tae",
	Short: "Tae é um utilitário CLI para extração e empacotamento de código",
	Long: `Tae (Tracker and Exporter) é uma ferramenta CLI modular para gerenciar, 
rastrear e extrair arquivos inteligentemente no disco.

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
	Example: `  # 1. Cria uma nova tag de agrupamento
  tae create "patch_v1"

  # 2. Adiciona arquivos à tag (suporta múltiplos alvos e filtro de exclusão)
  tae track ./src/main.go ./configs/ "patch_v1"
  tae track ./frontend/ --ignore "node_modules|*.log" "patch_v1"

  # 3. Verifica o que está sendo monitorado
  tae list "patch_v1"

  # 4. Exporta todos os arquivos agrupados para um diretório
  tae export "patch_v1" ./pasta_de_saida
  
  # 5. Exporta empacotando em .zip (fatiando a cada 1000 arquivos)
  tae export "patch_v1" ./pasta_de_saida --zip --limit 1000`,
}

func init() {
	// Desativa o comando "completion" automático (já documentamos isso no Long)
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	// Sobrescreve o template padrão do Cobra traduzindo os termos e tratando o comando "help"
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
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
