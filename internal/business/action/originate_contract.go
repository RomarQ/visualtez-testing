package action

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/romarq/visualtez-testing/internal/business"
	"github.com/romarq/visualtez-testing/internal/business/michelson"
	"github.com/romarq/visualtez-testing/internal/logger"
	"github.com/romarq/visualtez-testing/internal/utils"
)

type OriginateContractAction struct {
	Name    string          `json:"name"`
	Balance string          `json:"balance"`
	Code    json.RawMessage `json:"code"`
	Storage json.RawMessage `json:"storage"`
}

const (
	default_originator = "bootstrap2"
)

// Unmarshal action
func (action *OriginateContractAction) Unmarshal(bytes json.RawMessage) error {
	if err := json.Unmarshal(bytes, &action); err != nil {
		return err
	}

	// Validate action
	return action.validate()
}

// Perform action (Originates a contract)
func (action OriginateContractAction) Run(mockup business.Mockup) ActionResult {
	if mockup.ContainsAddress(action.Name) {
		return action.buildFailureResult(fmt.Sprintf("Name (%s) is already in use.", action.Name))
	}

	codeMicheline, err := michelson.MichelineOfJSON(action.Code)
	if err != nil {
		msg := fmt.Sprintf("Could not convert code from %s to %s.", business.JSON, business.Michelson)
		return action.buildFailureResult(msg)
	}
	storageMicheline, err := michelson.MichelineOfJSON(action.Storage)
	if err != nil {
		msg := fmt.Sprintf("Could not convert storage from %s to %s.", business.JSON, business.Michelson)
		return action.buildFailureResult(msg)
	}
	balance, ok := new(business.TMutez).SetString(action.Balance)
	if !ok {
		errMsg := fmt.Sprintf("invalid mutez value (%s).", action.Balance)
		logger.Debug("[Task #%s] - %s", mockup.TaskID, errMsg)
		return action.buildFailureResult(errMsg)
	}

	address, err := mockup.Originate(default_originator, action.Name, balance, codeMicheline, storageMicheline)
	if err != nil {
		logger.Debug("[Task #%s] - %s", mockup.TaskID, err)
		return action.buildFailureResult(fmt.Sprintf("could not originate contract. %s", err))
	}

	// Save new address
	mockup.SetAddress(action.Name, address)

	return action.buildSuccessResult(map[string]interface{}{
		"address": address,
	})
}

func (action OriginateContractAction) validate() error {
	missingFields := make([]string, 0)
	if action.Name == "" {
		missingFields = append(missingFields, "name")
	} else if err := utils.ValidateString(STRING_IDENTIFIER_REGEX, action.Name); err != nil {
		return err
	}
	if action.Code == nil {
		missingFields = append(missingFields, "code")
	}
	if action.Storage == nil {
		missingFields = append(missingFields, "storage")
	}

	if len(missingFields) > 0 {
		return fmt.Errorf("Action of kind (%s) misses the following fields [%s].", OriginateContract, strings.Join(missingFields, ", "))
	}

	return nil
}

func (action OriginateContractAction) buildSuccessResult(result map[string]interface{}) ActionResult {
	return ActionResult{
		Status: Success,
		Kind:   OriginateContract,
		Result: result,
		Action: action,
	}
}

func (action OriginateContractAction) buildFailureResult(details string) ActionResult {
	return ActionResult{
		Status: Failure,
		Kind:   OriginateContract,
		Action: action,
		Result: map[string]interface{}{
			"details": details,
		},
	}
}
