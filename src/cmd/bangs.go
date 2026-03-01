package cmd

import (
	"fmt"
	"strings"

	"github.com/cappuch/cuardach/src/bangs"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(bangsCmd)
}

var bangsCmd = &cobra.Command{
	Use:   "bangs",
	Short: "List available bang commands",
	Long:  "Bang commands redirect searches to specific sites. Use !w, !gh, !yt etc.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("  available bangs:")
		fmt.Println()
		for _, b := range bangs.List() {
			aliases := ""
			if len(b.Aliases) > 0 {
				aliases = " (" + strings.Join(b.Aliases, ", ") + ")"
			}
			fmt.Printf("  !%-5s  %-18s%s\n", b.Name, b.Title, aliases)
		}
		fmt.Println()
		fmt.Println("  usage: cuardach search \"!w rust\" or cuardach search \"rust !gh\"")
	},
}
