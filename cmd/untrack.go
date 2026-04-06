package cmd

import (
	"fmt"
	"os"

	"tae/internal/storage"

	"github.com/spf13/cobra"
)

var untrackCmd = &cobra.Command{
	Use:   "untrack <arquivo1> [arquivo2...] <nome da tag>",
	Short: "Remove um ou mais arquivos/diretórios do monitoramento de uma tag",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		tagName := args[len(args)-1]
		targets := args[:len(args)-1]

		for _, target := range targets {
			if err := storage.UntrackPath(tagName, target); err != nil {
				fmt.Fprintf(os.Stderr, "Erro ao remover '%s': %v\n", target, err)
			} else {
				fmt.Printf("Alvo '%s' removido da tag '%s'.\n", target, tagName)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(untrackCmd)
}
