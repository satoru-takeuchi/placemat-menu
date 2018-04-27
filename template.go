package menu


// Rack is template args for rack
type Rack struct {
  Name string
}

// Spine is template args for Spine
type Spine struct {
  Name string
}

// TemplateArgs is args for cluster.yml
type TemplateArgs struct {
  Network struct {
    External string
  }
  Racks []Rack
  Spines []Spine
}

// MenuToTemplateArgs is converter Menu to TemplateArgs 
func MenuToTemplateArgs(menu Menu) (TemplateArgs, error) {
  var templateArgs TemplateArgs

  return templateArgs, nil
}

