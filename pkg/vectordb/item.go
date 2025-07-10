// Package vectordb provides vector database implementations for MemGOS
package vectordb

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// VectorDBItem represents a vector database item with ID, vector, payload, and optional score
type VectorDBItem struct {
	ID      string                 `json:"id"`
	Vector  []float32              `json:"vector,omitempty"`
	Payload map[string]interface{} `json:"payload,omitempty"`
	Score   *float32               `json:"score,omitempty"`
}

// NewVectorDBItem creates a new VectorDBItem with validation
func NewVectorDBItem(id string, vector []float32, payload map[string]interface{}) (*VectorDBItem, error) {
	// Generate UUID if ID is empty
	if id == "" {
		id = uuid.New().String()
	}

	// Validate ID format (should be UUID)
	if !isValidUUID(id) {
		return nil, fmt.Errorf("invalid ID format: %s, must be a valid UUID", id)
	}

	return &VectorDBItem{
		ID:      id,
		Vector:  vector,
		Payload: payload,
	}, nil
}

// NewVectorDBItemWithScore creates a new VectorDBItem with a score (for search results)
func NewVectorDBItemWithScore(id string, vector []float32, payload map[string]interface{}, score float32) (*VectorDBItem, error) {
	item, err := NewVectorDBItem(id, vector, payload)
	if err != nil {
		return nil, err
	}
	item.Score = &score
	return item, nil
}

// Validate validates the VectorDBItem
func (v *VectorDBItem) Validate() error {
	if v.ID == "" {
		return fmt.Errorf("ID cannot be empty")
	}
	
	if !isValidUUID(v.ID) {
		return fmt.Errorf("invalid ID format: %s, must be a valid UUID", v.ID)
	}
	
	return nil
}

// ToMap converts VectorDBItem to a map representation
func (v *VectorDBItem) ToMap() map[string]interface{} {
	result := map[string]interface{}{
		"id": v.ID,
	}
	
	if v.Vector != nil {
		result["vector"] = v.Vector
	}
	
	if v.Payload != nil {
		result["payload"] = v.Payload
	}
	
	if v.Score != nil {
		result["score"] = *v.Score
	}
	
	return result
}

// FromMap creates a VectorDBItem from a map representation
func FromMap(data map[string]interface{}) (*VectorDBItem, error) {
	item := &VectorDBItem{}
	
	// Extract ID
	if id, ok := data["id"].(string); ok {
		item.ID = id
	} else {
		return nil, fmt.Errorf("missing or invalid ID field")
	}
	
	// Extract Vector
	if vectorData, ok := data["vector"]; ok {
		switch v := vectorData.(type) {
		case []float32:
			item.Vector = v
		case []interface{}:
			vector := make([]float32, len(v))
			for i, val := range v {
				if f, ok := val.(float64); ok {
					vector[i] = float32(f)
				} else if f, ok := val.(float32); ok {
					vector[i] = f
				} else {
					return nil, fmt.Errorf("invalid vector element type at index %d", i)
				}
			}
			item.Vector = vector
		}
	}
	
	// Extract Payload
	if payload, ok := data["payload"].(map[string]interface{}); ok {
		item.Payload = payload
	}
	
	// Extract Score
	if score, ok := data["score"]; ok {
		switch s := score.(type) {
		case float32:
			item.Score = &s
		case float64:
			scoreFloat32 := float32(s)
			item.Score = &scoreFloat32
		}
	}
	
	return item, item.Validate()
}

// Clone creates a deep copy of the VectorDBItem
func (v *VectorDBItem) Clone() *VectorDBItem {
	item := &VectorDBItem{
		ID: v.ID,
	}
	
	// Clone vector
	if v.Vector != nil {
		item.Vector = make([]float32, len(v.Vector))
		copy(item.Vector, v.Vector)
	}
	
	// Clone payload
	if v.Payload != nil {
		item.Payload = make(map[string]interface{})
		for k, val := range v.Payload {
			item.Payload[k] = val
		}
	}
	
	// Clone score
	if v.Score != nil {
		score := *v.Score
		item.Score = &score
	}
	
	return item
}

// String returns a string representation of the VectorDBItem
func (v *VectorDBItem) String() string {
	var parts []string
	parts = append(parts, fmt.Sprintf("ID: %s", v.ID))
	
	if v.Vector != nil {
		parts = append(parts, fmt.Sprintf("Vector: [%d dimensions]", len(v.Vector)))
	}
	
	if v.Payload != nil {
		parts = append(parts, fmt.Sprintf("Payload: %d fields", len(v.Payload)))
	}
	
	if v.Score != nil {
		parts = append(parts, fmt.Sprintf("Score: %.4f", *v.Score))
	}
	
	return fmt.Sprintf("VectorDBItem{%s}", strings.Join(parts, ", "))
}

// MarshalJSON implements custom JSON marshaling
func (v *VectorDBItem) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.ToMap())
}

// UnmarshalJSON implements custom JSON unmarshaling
func (v *VectorDBItem) UnmarshalJSON(data []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	
	item, err := FromMap(raw)
	if err != nil {
		return err
	}
	
	*v = *item
	return nil
}

// isValidUUID validates if a string is a valid UUID
func isValidUUID(id string) bool {
	_, err := uuid.Parse(id)
	return err == nil
}

// VectorDBItemList represents a list of VectorDBItems with utility methods
type VectorDBItemList []*VectorDBItem

// Len returns the length of the list
func (list VectorDBItemList) Len() int {
	return len(list)
}

// IDs returns a slice of all IDs in the list
func (list VectorDBItemList) IDs() []string {
	ids := make([]string, len(list))
	for i, item := range list {
		ids[i] = item.ID
	}
	return ids
}

// FindByID finds an item by ID in the list
func (list VectorDBItemList) FindByID(id string) *VectorDBItem {
	for _, item := range list {
		if item.ID == id {
			return item
		}
	}
	return nil
}

// Filter filters items based on a predicate function
func (list VectorDBItemList) Filter(predicate func(*VectorDBItem) bool) VectorDBItemList {
	var filtered VectorDBItemList
	for _, item := range list {
		if predicate(item) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

// ToMaps converts the list to a slice of maps
func (list VectorDBItemList) ToMaps() []map[string]interface{} {
	maps := make([]map[string]interface{}, len(list))
	for i, item := range list {
		maps[i] = item.ToMap()
	}
	return maps
}