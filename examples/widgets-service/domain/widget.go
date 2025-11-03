package domain

// Widget is the domain aggregate representing a widget resource.
type Widget struct {
	id          WidgetID
	name        string
	description string
	ownerID     string
}

// NewWidget creates a new Widget with the given attributes.
func NewWidget(id WidgetID, name, description, ownerID string) *Widget {
	return &Widget{
		id:          id,
		name:        name,
		description: description,
		ownerID:     ownerID,
	}
}

// ID returns the widget's identifier.
func (w *Widget) ID() WidgetID {
	return w.id
}

// Name returns the widget's name.
func (w *Widget) Name() string {
	return w.name
}

// Description returns the widget's description.
func (w *Widget) Description() string {
	return w.description
}

// OwnerID returns the ID of the widget's owner.
func (w *Widget) OwnerID() string {
	return w.ownerID
}

// Serialize converts the Widget to a serializable representation.
func (w *Widget) Serialize() WidgetData {
	return WidgetData{
		ID:          string(w.id),
		Name:        w.name,
		Description: w.description,
		OwnerID:     w.ownerID,
	}
}

// DeserializeWidget converts serialized data back to a Widget domain object.
func DeserializeWidget(data WidgetData) *Widget {
	return &Widget{
		id:          WidgetID(data.ID),
		name:        data.Name,
		description: data.Description,
		ownerID:     data.OwnerID,
	}
}

// WidgetData is the serializable representation of a Widget.
type WidgetData struct {
	ID          string
	Name        string
	Description string
	OwnerID     string
}

