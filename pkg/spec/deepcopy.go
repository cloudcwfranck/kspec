package spec

// DeepCopyInto is a manually written deepcopy function for SpecFields.
func (in *SpecFields) DeepCopyInto(out *SpecFields) {
	*out = *in
	if in.PodSecurity != nil {
		in, out := &in.PodSecurity, &out.PodSecurity
		*out = new(PodSecuritySpec)
		(*in).DeepCopyInto(*out)
	}
	if in.Network != nil {
		in, out := &in.Network, &out.Network
		*out = new(NetworkSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.Workloads != nil {
		in, out := &in.Workloads, &out.Workloads
		*out = new(WorkloadsSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.RBAC != nil {
		in, out := &in.RBAC, &out.RBAC
		*out = new(RBACSpec)
		**out = **in
	}
	if in.Admission != nil {
		in, out := &in.Admission, &out.Admission
		*out = new(AdmissionSpec)
		**out = **in
	}
	if in.Observability != nil {
		in, out := &in.Observability, &out.Observability
		*out = new(ObservabilitySpec)
		**out = **in
	}
	if in.Compliance != nil {
		in, out := &in.Compliance, &out.Compliance
		*out = new(ComplianceSpec)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is a manually written deepcopy function for SpecFields.
func (in *SpecFields) DeepCopy() *SpecFields {
	if in == nil {
		return nil
	}
	out := new(SpecFields)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto for PodSecuritySpec
func (in *PodSecuritySpec) DeepCopyInto(out *PodSecuritySpec) {
	*out = *in
	if in.Exemptions != nil {
		in, out := &in.Exemptions, &out.Exemptions
		*out = make([]PodSecurityExemption, len(*in))
		copy(*out, *in)
	}
}

// DeepCopyInto for NetworkSpec
func (in *NetworkSpec) DeepCopyInto(out *NetworkSpec) {
	*out = *in
}

// DeepCopyInto for WorkloadsSpec
func (in *WorkloadsSpec) DeepCopyInto(out *WorkloadsSpec) {
	*out = *in
	if in.Containers != nil {
		in, out := &in.Containers, &out.Containers
		*out = new(ContainerSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.Images != nil {
		in, out := &in.Images, &out.Images
		*out = new(ImageSpec)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopyInto for ContainerSpec
func (in *ContainerSpec) DeepCopyInto(out *ContainerSpec) {
	*out = *in
	if in.Required != nil {
		in, out := &in.Required, &out.Required
		*out = make([]FieldRequirement, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Forbidden != nil {
		in, out := &in.Forbidden, &out.Forbidden
		*out = make([]FieldRequirement, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopyInto for FieldRequirement (handles interface{})
func (in *FieldRequirement) DeepCopyInto(out *FieldRequirement) {
	*out = *in
	if in.Exists != nil {
		in, out := &in.Exists, &out.Exists
		*out = new(bool)
		**out = **in
	}
	// Note: Value is interface{} and cannot be deep copied automatically
	// For safety, we just copy the reference. In practice, this is fine
	// because FieldRequirement values are typically primitives (bool, string, int)
	// which are immutable in Go.
}

// DeepCopyInto for ImageSpec
func (in *ImageSpec) DeepCopyInto(out *ImageSpec) {
	*out = *in
	if in.AllowedRegistries != nil {
		in, out := &in.AllowedRegistries, &out.AllowedRegistries
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.BlockedRegistries != nil {
		in, out := &in.BlockedRegistries, &out.BlockedRegistries
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopyInto for ComplianceSpec
func (in *ComplianceSpec) DeepCopyInto(out *ComplianceSpec) {
	*out = *in
	if in.Frameworks != nil {
		in, out := &in.Frameworks, &out.Frameworks
		*out = make([]ComplianceFramework, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopyInto for ComplianceFramework
func (in *ComplianceFramework) DeepCopyInto(out *ComplianceFramework) {
	*out = *in
	if in.Controls != nil {
		in, out := &in.Controls, &out.Controls
		*out = make([]ComplianceControl, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopyInto for ComplianceControl
func (in *ComplianceControl) DeepCopyInto(out *ComplianceControl) {
	*out = *in
	if in.Mappings != nil {
		in, out := &in.Mappings, &out.Mappings
		*out = make([]ControlMapping, len(*in))
		copy(*out, *in)
	}
}
