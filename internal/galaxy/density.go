package galaxy

import "math"

// densityField computes the combined stellar density ρ(x,y,z) for galaxy generation.
// All coordinates are in light-years.
type densityField struct {
	// Disk parameters
	scaleLength float64 // R_d ≈ 3500 ly
	scaleHeight float64 // h_z ≈ 1000 ly

	// Bulge/bar parameters
	bulgeRadius   float64 // effective bulge radius
	barElongation float64 // how much the bar stretches along x vs y

	// Spiral arm parameters
	arms       int
	armWinding float64 // b: R(θ) = R_bar * exp(b*θ)
	armSpread  float64 // σ_arm: angular width of arm boost (radians)
	barRadius  float64 // inner radius where arms start (ly)
}

// newDensityField constructs a density field from generator config.
func newDensityField(arms int, armWinding, armSpread, radiusLY float64) *densityField {
	return &densityField{
		scaleLength:   radiusLY * 0.07, // R_d ≈ 7% of galaxy radius
		scaleHeight:   1000,
		bulgeRadius:   radiusLY * 0.04,
		barElongation: 2.0, // bar is 2× longer along x than y
		arms:          arms,
		armWinding:    armWinding,
		armSpread:     armSpread,
		barRadius:     radiusLY * 0.08, // arms start at 8% of radius
	}
}

// Evaluate returns the unnormalized density at (x, y, z) in light-years.
// The result is the sum of disk + bulge + arm components.
func (d *densityField) Evaluate(x, y, z float64) float64 {
	R := math.Sqrt(x*x + y*y) // cylindrical radius
	return d.disk(R, z) + d.bulge(x, y, z) + d.arms_(R, x, y)
}

// disk returns the exponential disk density component.
func (d *densityField) disk(R, z float64) float64 {
	return math.Exp(-R/d.scaleLength) * math.Exp(-math.Abs(z)/d.scaleHeight)
}

// bulge returns the de Vaucouleurs bulge density with bar elongation.
// The bar stretches along the x-axis.
func (d *densityField) bulge(x, y, z float64) float64 {
	// Squash y-axis to simulate bar elongation
	xBar := x / d.barElongation
	r := math.Sqrt(xBar*xBar + y*y + z*z)
	if r < 1 {
		r = 1
	}
	// de Vaucouleurs profile: ρ ∝ exp(-b * (r/r_e)^(1/4))
	// b ≈ 7.67 normalisation constant
	re := d.bulgeRadius
	return 2.0 * math.Exp(-7.67*(math.Pow(r/re, 0.25)-1))
}

// arms_ returns the spiral arm density boost at cylindrical coordinates (R, x, y).
func (d *densityField) arms_(R, x, y float64) float64 {
	if R < d.barRadius*0.5 {
		return 0 // inside bar core, no arm contribution
	}

	theta := math.Atan2(y, x) // azimuthal angle [-π, π]
	boost := 0.0

	for arm := range d.arms {
		// Each arm starts at angle offset = 2π/arms * arm
		thetaStart := 2 * math.Pi / float64(d.arms) * float64(arm)

		// Logarithmic spiral: the arm angle at radius R
		// R = R_bar * exp(b * (theta_arm - thetaStart))
		// → theta_arm = thetaStart + ln(R/R_bar) / b
		if R <= 0 || d.barRadius <= 0 || d.armWinding <= 0 {
			continue
		}
		thetaArm := thetaStart + math.Log(R/d.barRadius)/d.armWinding

		// Circular angular distance
		dTheta := theta - thetaArm
		// Normalize to [-π, π]
		dTheta = math.Mod(dTheta+math.Pi, 2*math.Pi) - math.Pi

		sigma := d.armSpread
		armBoost := math.Exp(-dTheta*dTheta / (2 * sigma * sigma))

		// Radial envelope: arm strength peaks at mid-radius and fades at edges
		radialEnv := math.Exp(-math.Pow((R-d.barRadius*3)/(d.barRadius*4), 2))
		boost += armBoost * radialEnv
	}

	return boost
}

// maxDensity estimates the peak density for rejection sampling normalization.
// Returns a value slightly above the true maximum.
func (d *densityField) maxDensity() float64 {
	// Bulge center is the global maximum; we add 10% margin.
	return (d.disk(0, 0) + d.bulge(0, 0, 0) + float64(d.arms)*1.0) * 1.1
}
