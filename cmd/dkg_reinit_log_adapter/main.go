package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/lidofinance/dc4bc/client/operations"

	"github.com/lidofinance/dc4bc/client"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "dkg_reinit_log_adpater",
	Short: "DKG reinit log adpater",
}

// "id": "6d285758-8bde-4925-ba72-217650063bd1",
// "dkg_round_id": "37f16666f014026c4e96cf84868f8cfb",
// "offset": 0,
// "event": "event_sig_proposal_init",
// "data": "eyJQYXJ0aWNpcGFudHMiOlt7IlVzZXJuYW1lIjoibm9kZV8wIiwiUHViS2V5IjoiRHV4M1RtSEptSXphQmZsbHhSSmFCTHIvYy9uWWFraUdZelBhaC9peTJVQT0iLCJEa2dQdWJLZXkiOiJoVVJ6THB4OW5RSkVVekhQcTJXYnJFcHNKdVF3KzBnV0o1cGtDRkRRd0prNXlwM2JCbUtES1Z5c3cyTjNxa3hJIn0seyJVc2VybmFtZSI6Im5vZGVfMSIsIlB1YktleSI6ImZwRzRGS0xBNGpwN3JHMkhzcnFmYURhRFUyRWJCMzdabFNSNUZKdEdtU1E9IiwiRGtnUHViS2V5IjoiaGFoWXAweHkwTWtITXpWVU9MdjRYMjBTbHFBc3drT2dpQklVMFJ1aFZ6N2FKaWorOGZ0QjJHeFlPL0JWNk4yQiJ9LHsiVXNlcm5hbWUiOiJub2RlXzIiLCJQdWJLZXkiOiIxN2Z5SkE2Uk5qa0xZWTlvUElBaTZnVEUwUzdWOTV4aHlZc2FXODQvWnJ3PSIsIkRrZ1B1YktleSI6InBkUkJRREphU0tQcXVaZ0NXeWlSazVWRlFDeE9VSjI0Unp0MGk2OGpVY05keFJtRzRJbmcxN3FqYzJsTll3a2QifSx7IlVzZXJuYW1lIjoibm9kZV8zIiwiUHViS2V5IjoiUEZvb0p5blBwUnp4UG1FYVRpSlc3a1REVVczdXBseEUzeUVNY2VGSHl1Zz0iLCJEa2dQdWJLZXkiOiJpRXJWTXM1WXFFc1EzSG5HelQ0N0lIOWNnd3BTZUFZT3NobGtUNngzSW4zVnZ0eURrR1NIVjBNbnYwZ1poZFcyIn1dLCJTaWduaW5nVGhyZXNob2xkIjoyLCJDcmVhdGVkQXQiOiIyMDIxLTA3LTA1VDEyOjEzOjI1LjAxNzcxMzc0NCswMzowMCJ9",
// "signature": "Jt8ooobkFE5ilOruF7RONvA49eUe6PYewwqG1JD/wlX7B0WsAvpvnoSPbWTcdkXMmOJhjErP1LpukHe4y8faBA==",
// "sender": "node_3",
// "recipient": ""
// },

func adapt() *cobra.Command {
	return &cobra.Command{
		Use:   "adapt",
		Short: "reads a DKG reinit JSON created by release 0.1.4 and apapt it for latest dc4bc.",

		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			inputFile := args[0]
			outputFile := args[1]

			data, err := ioutil.ReadFile(inputFile)
			if err != nil {
				return fmt.Errorf("failed to read input file: %v", err)
			}
			var reDKG operations.ReDKG

			err = json.Unmarshal(data, &reDKG)
			if err != nil {
				return fmt.Errorf("failed to decode data into reDKG: %v", err)
			}

			adaptedReDKG, err := client.GetAdaptedReDKG(reDKG)
			if err != nil {
				return fmt.Errorf("failed to adapt reinit DKG message: %v", err)
			}
			reDKGBz, err := json.Marshal(adaptedReDKG)
			if err != nil {
				return fmt.Errorf("failed to encode adapted reinit DKG message: %v", err)
			}

			if err = ioutil.WriteFile(outputFile, reDKGBz, 0666); err != nil {
				return fmt.Errorf("failed to save adapted reinit DKG JSON: %v", err)
			}
			return nil
		},
	}
}

func main() {
	rootCmd.AddCommand(
		adapt(),
	)
	rootCmd.SetUsageTemplate("dkg_reinit_log_adpater adapt <input_file_name> <output_file_name>")
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("Failed to execute root command: %v", err)
	}
}
