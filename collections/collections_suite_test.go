package collections_test

import (
	"testing"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestCollections(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Collections Suite")
}

type datastore interface {
	clear() int
}

type Org struct {
	ID       uuid.UUID  `json:"id"`
	Name     string     `json:"name"`
	ParentID *uuid.UUID `json:"parentID"`
}

func (o Org) Identifier() uuid.UUID {
	return o.ID
}

func (o Org) ParentIdentifier() *uuid.UUID {
	return o.ParentID
}

var unitedNations []Org // Declare orgs as an empty slice

func init() {
	unitedNations = append(unitedNations, Org{ID: uuid.New(), Name: "UN", ParentID: nil})                           // United Nations
	unitedNations = append(unitedNations, Org{ID: uuid.New(), Name: "UNICEF", ParentID: &unitedNations[0].ID})      // United Nations Children's Fund
	unitedNations = append(unitedNations, Org{ID: uuid.New(), Name: "UNESCO", ParentID: &unitedNations[0].ID})      // United Nations Educational, Scientific and Cultural Organization
	unitedNations = append(unitedNations, Org{ID: uuid.New(), Name: "WHO", ParentID: &unitedNations[0].ID})         // World Health Organization
	unitedNations = append(unitedNations, Org{ID: uuid.New(), Name: "UNDP", ParentID: &unitedNations[0].ID})        // United Nations Development Programme
	unitedNations = append(unitedNations, Org{ID: uuid.New(), Name: "UNHCR", ParentID: &unitedNations[0].ID})       // United Nations High Commissioner for Refugees
	unitedNations = append(unitedNations, Org{ID: uuid.New(), Name: "UNESCO", ParentID: &unitedNations[0].ID})      // United Nations Educational, Scientific and Cultural Organization
	unitedNations = append(unitedNations, Org{ID: uuid.New(), Name: "UNEP", ParentID: &unitedNations[0].ID})        // United Nations Environment Programme
	unitedNations = append(unitedNations, Org{ID: uuid.New(), Name: "UNODC", ParentID: &unitedNations[0].ID})       // United Nations Office on Drugs and Crime
	unitedNations = append(unitedNations, Org{ID: uuid.New(), Name: "UNW", ParentID: &unitedNations[0].ID})         // United Nations Entity for Gender Equality and the Empowerment of Women
	unitedNations = append(unitedNations, Org{ID: uuid.New(), Name: "UNH", ParentID: &unitedNations[0].ID})         // United Nations Human Settlements Programme
	unitedNations = append(unitedNations, Org{ID: uuid.New(), Name: "UNIDO", ParentID: &unitedNations[0].ID})       // United Nations Industrial Development Organization
	unitedNations = append(unitedNations, Org{ID: uuid.New(), Name: "UNFPA", ParentID: &unitedNations[0].ID})       // United Nations Population Fund
	unitedNations = append(unitedNations, Org{ID: uuid.New(), Name: "UNOPS", ParentID: &unitedNations[0].ID})       // United Nations Office for Project Services
	unitedNations = append(unitedNations, Org{ID: uuid.New(), Name: "UNWTO", ParentID: &unitedNations[0].ID})       // United Nations World Tourism Organization
	unitedNations = append(unitedNations, Org{ID: uuid.New(), Name: "UNRWA", ParentID: &unitedNations[0].ID})       // United Nations Relief and Works Agency for Palestine Refugees in the Near East
	unitedNations = append(unitedNations, Org{ID: uuid.New(), Name: "UNESCO", ParentID: &unitedNations[0].ID})      // United Nations Educational, Scientific and Cultural Organization
	unitedNations = append(unitedNations, Org{ID: uuid.New(), Name: "UNEP", ParentID: &unitedNations[0].ID})        // United Nations Environment Programme
	unitedNations = append(unitedNations, Org{ID: uuid.New(), Name: "UNODC", ParentID: &unitedNations[0].ID})       // United Nations Office on Drugs and Crime
	unitedNations = append(unitedNations, Org{ID: uuid.New(), Name: "UNW", ParentID: &unitedNations[0].ID})         // United Nations Entity for Gender Equality and the Empowerment of Women
	unitedNations = append(unitedNations, Org{ID: uuid.New(), Name: "UNH", ParentID: &unitedNations[0].ID})         // United Nations Human Settlements Programme
	unitedNations = append(unitedNations, Org{ID: uuid.New(), Name: "UNIDO", ParentID: &unitedNations[0].ID})       // United Nations Industrial Development Organization
	unitedNations = append(unitedNations, Org{ID: uuid.New(), Name: "UNFPA", ParentID: &unitedNations[0].ID})       // United Nations Population Fund
	unitedNations = append(unitedNations, Org{ID: uuid.New(), Name: "UNOPS", ParentID: &unitedNations[0].ID})       // United Nations Office for Project Services
	unitedNations = append(unitedNations, Org{ID: uuid.New(), Name: "UNWTO", ParentID: &unitedNations[0].ID})       // United Nations World Tourism Organization
	unitedNations = append(unitedNations, Org{ID: uuid.New(), Name: "UNRWA", ParentID: &unitedNations[0].ID})       // United Nations Relief and Works Agency for Palestine Refugees in the Near East
	unitedNations = append(unitedNations, Org{ID: uuid.New(), Name: "AU", ParentID: &unitedNations[0].ID})          // African Union
	unitedNations = append(unitedNations, Org{ID: uuid.New(), Name: "AUC", ParentID: &unitedNations[26].ID})        // African Union Commission
	unitedNations = append(unitedNations, Org{ID: uuid.New(), Name: "AUP", ParentID: &unitedNations[26].ID})        // African Union Parliament
	unitedNations = append(unitedNations, Org{ID: uuid.New(), Name: "AUPSC", ParentID: &unitedNations[26].ID})      // African Union Peace and Security Council
	unitedNations = append(unitedNations, Org{ID: uuid.New(), Name: "ECOWAS", ParentID: &unitedNations[26].ID})     // Economic Community of West African States
	unitedNations = append(unitedNations, Org{ID: uuid.New(), Name: "EAC", ParentID: &unitedNations[26].ID})        // East African Community
	unitedNations = append(unitedNations, Org{ID: uuid.New(), Name: "SADC", ParentID: &unitedNations[26].ID})       // Southern African Development Community
	unitedNations = append(unitedNations, Org{ID: uuid.New(), Name: "COMESA", ParentID: &unitedNations[26].ID})     // Common Market for Eastern and Southern Africa
	unitedNations = append(unitedNations, Org{ID: uuid.New(), Name: "IGAD", ParentID: &unitedNations[26].ID})       // Intergovernmental Authority on Development
	unitedNations = append(unitedNations, Org{ID: uuid.New(), Name: "NEPAD", ParentID: &unitedNations[26].ID})      // New Partnership for Africa's Development
	unitedNations = append(unitedNations, Org{ID: uuid.New(), Name: "PAP", ParentID: &unitedNations[26].ID})        // Pan-African Parliament
	unitedNations = append(unitedNations, Org{ID: uuid.New(), Name: "AU-IBAR", ParentID: &unitedNations[26].ID})    // African Union - Interafrican Bureau for Animal Resources
	unitedNations = append(unitedNations, Org{ID: uuid.New(), Name: "AUDA-NEPAD", ParentID: &unitedNations[26].ID}) // African Union Development Agency - New Partnership for Africa's Development
} 
