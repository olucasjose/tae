package cmd

import (
	"fmt"
	"os"

	"tae/internal/storage"

	"github.com/spf13/cobra"
)

var untrackCmd = &cobra.Command{
	Use:   "untrack <caminho>",
	Short: "Remove um arquivo ou diretório do monitoramento de uma tag",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		target := args[0]

		if err := storage.UntrackPath(tagFlag, target); err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao remover alvo: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Alvo '%s' removido da tag '%s'.\n", target, tagFlag)
	},
}

func init() {
	untrackCmd.Flags().StringVarP(&tagFlag, "tag", "t", "", "Nome da tag (obrigatório)")
	untrackCmd.MarkFlagRequired("tag")
	rootCmd.AddCommand(untrackCmd)
}
