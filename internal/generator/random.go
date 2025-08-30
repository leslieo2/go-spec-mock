package generator

import (
	"crypto/rand"
	"math"
	"math/big"
	"time"

	"github.com/go-faker/faker/v4"
	"github.com/lucasjones/reggen"
)

// RandomSource provides a unified interface for all random operations
type RandomSource interface {
	// Basic random operations
	Intn(n int) int
	Float64() float64
	Int() int

	// Faker-like operations
	Email() string
	FirstName() string
	LastName() string
	Name() string
	Username() string
	Phonenumber() string
	Sentence() string
	Word() string
	UUIDHyphenated() string
	URL() string
	DomainName() string
	IPv4() string
	IPv6() string

	// Pattern generation
	GeneratePattern(pattern string, maxLength int) (string, error)

	// Date operations
	Date() string
	DateTime() string
}

// SecureRandomSource implements RandomSource with cryptographically secure randomness
type SecureRandomSource struct{}

// NewSecureRandomSource creates a new secure random source
func NewSecureRandomSource() *SecureRandomSource {
	return &SecureRandomSource{}
}

func (s *SecureRandomSource) Intn(n int) int {
	if n <= 0 {
		return 0
	}
	val, _ := rand.Int(rand.Reader, big.NewInt(int64(n)))
	return int(val.Int64())
}

func (s *SecureRandomSource) Float64() float64 {
	val, _ := rand.Int(rand.Reader, big.NewInt(1<<53))
	return float64(val.Int64()) / (1 << 53)
}

func (s *SecureRandomSource) Int() int {
	val, _ := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	return int(val.Int64())
}

func (s *SecureRandomSource) Email() string {
	return faker.Email()
}

func (s *SecureRandomSource) FirstName() string {
	return faker.FirstName()
}

func (s *SecureRandomSource) LastName() string {
	return faker.LastName()
}

func (s *SecureRandomSource) Name() string {
	return faker.Name()
}

func (s *SecureRandomSource) Username() string {
	return faker.Username()
}

func (s *SecureRandomSource) Phonenumber() string {
	return faker.Phonenumber()
}

func (s *SecureRandomSource) Sentence() string {
	return faker.Sentence()
}

func (s *SecureRandomSource) Word() string {
	return faker.Word()
}

func (s *SecureRandomSource) UUIDHyphenated() string {
	return faker.UUIDHyphenated()
}

func (s *SecureRandomSource) URL() string {
	return faker.URL()
}

func (s *SecureRandomSource) DomainName() string {
	return faker.DomainName()
}

func (s *SecureRandomSource) IPv4() string {
	return faker.IPv4()
}

func (s *SecureRandomSource) IPv6() string {
	return faker.IPv6()
}

func (s *SecureRandomSource) GeneratePattern(pattern string, maxLength int) (string, error) {
	return reggen.Generate(pattern, maxLength)
}

func (s *SecureRandomSource) Date() string {
	days := s.Intn(365)
	date := time.Now().AddDate(0, 0, days-182) // ±6 months from now
	return date.Format("2006-01-02")
}

func (s *SecureRandomSource) DateTime() string {
	days := s.Intn(365)
	datetime := time.Now().AddDate(0, 0, days-182) // ±6 months from now
	return datetime.Format(time.RFC3339)
}
