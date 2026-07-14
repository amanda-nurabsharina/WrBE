package controller

import (
	"app/src/model"
	"app/src/response"
	"app/src/service"
	"app/src/validation"

	"github.com/gofiber/fiber/v2"
)

type ProductController struct {
	ProductService service.ProductService
}

func NewProductController(productService service.ProductService) *ProductController {
	return &ProductController{
		ProductService: productService,
	}
}

func (ctrl *ProductController) GetProducts(c *fiber.Ctx) error {
	search := c.Query("search", "")
	products, err := ctrl.ProductService.GetProducts(c, search)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).
		JSON(response.SuccessWithData[[]model.Product]{
			Code:    fiber.StatusOK,
			Status:  "success",
			Message: "Products retrieved successfully",
			Data:    products,
		})
}

func (ctrl *ProductController) GetProductByID(c *fiber.Ctx) error {
	id := c.Params("id")
	product, err := ctrl.ProductService.GetProductByID(c, id)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).
		JSON(response.SuccessWithData[*model.Product]{
			Code:    fiber.StatusOK,
			Status:  "success",
			Message: "Product retrieved successfully",
			Data:    product,
		})
}

func (ctrl *ProductController) CreateProduct(c *fiber.Ctx) error {
	req := new(validation.CreateProduct)
	if err := c.BodyParser(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	product, err := ctrl.ProductService.CreateProduct(c, req)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).
		JSON(response.SuccessWithData[*model.Product]{
			Code:    fiber.StatusCreated,
			Status:  "success",
			Message: "Product created successfully",
			Data:    product,
		})
}

func (ctrl *ProductController) UpdateProduct(c *fiber.Ctx) error {
	id := c.Params("id")
	req := new(validation.UpdateProduct)
	if err := c.BodyParser(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	product, err := ctrl.ProductService.UpdateProduct(c, id, req)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).
		JSON(response.SuccessWithData[*model.Product]{
			Code:    fiber.StatusOK,
			Status:  "success",
			Message: "Product updated successfully",
			Data:    product,
		})
}

func (ctrl *ProductController) DeleteProduct(c *fiber.Ctx) error {
	id := c.Params("id")
	err := ctrl.ProductService.DeleteProduct(c, id)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).
		JSON(response.Common{
			Code:    fiber.StatusOK,
			Status:  "success",
			Message: "Product deleted successfully",
		})
}
