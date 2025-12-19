package product

type Service interface {
	CreateProduct(req CreateProductRequest) (*Product, error)
	GetAllProducts() ([]Product, error)
	GetProductByID(id uint64) (*Product, error)
	UpdateProduct(id uint64, req CreateProductRequest) (*Product, error)
	DeleteProduct(id uint64) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) CreateProduct(req CreateProductRequest) (*Product, error) {
	newProduct := &Product{
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Stock:       req.Stock,
	}
	err := s.repo.Create(newProduct)
	if err != nil {
		return nil, err
	}
	return newProduct, err
}

func (s *service) GetAllProducts() ([]Product, error) {
	return s.repo.FindAll()
}

func (s *service) GetProductByID(id uint64) (*Product, error) {
	return s.repo.FindByID(id)
}

func (s *service) UpdateProduct(id uint64, req CreateProductRequest) (*Product, error) {
	product, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	product.Name = req.Name
	product.Description = req.Description
	product.Price = req.Price
	product.Stock = req.Stock
	err = s.repo.Update(product)
	if err != nil {
		return nil, err
	}
	return product, nil
}
func (s *service) DeleteProduct(id uint64) error {
	return s.repo.Delete(id)
}
