package cmd

import (
	"fmt"
	"os"

	"tae/internal/storage"

	"github.com/spf13/cobra"
)

var tagFlag string

var trackCmd = &cobra.Command{
	Use:   "track <caminho>",
	Short: "Adiciona um arquivo ou diretório ao monitoramento de uma tag",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		target := args[0]

		if _, err := os.Stat(target); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Erro: O alvo '%s' não existe no disco.\n", target)
			os.Exit(1)
		}

		if err := storage.TrackPath(tagFlag, target); err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao rastrear alvo: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Alvo '%s' rastreado com sucesso na tag '%s'.\n", target, tagFlag)
	},
}

func init() {
	trackCmd.Flags().StringVarP(&tagFlag, "tag", "t", "", "Nome da tag (obrigatório)")
	trackCmd.MarkFlagRequired("tag")
	rootCmd.AddCommand(trackCmd)
}
