package controller

import (
	"app/src/model"
	"app/src/response"
	"app/src/service"
	"app/src/validation"

	"github.com/gofiber/fiber/v2"
)

type CustomerController struct {
	CustomerService service.CustomerService
}

func NewCustomerController(customerService service.CustomerService) *CustomerController {
	return &CustomerController{
		CustomerService: customerService,
	}
}

func (ctrl *CustomerController) GetCustomers(c *fiber.Ctx) error {
	search := c.Query("search", "")
	list, err := ctrl.CustomerService.GetCustomers(c, search)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).
		JSON(response.SuccessWithData[[]model.Customer]{
			Code:    fiber.StatusOK,
			Status:  "success",
			Message: "Customers retrieved successfully",
			Data:    list,
		})
}

func (ctrl *CustomerController) GetCustomerByID(c *fiber.Ctx) error {
	id := c.Params("id")
	customer, err := ctrl.CustomerService.GetCustomerByID(c, id)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).
		JSON(response.SuccessWithData[*model.Customer]{
			Code:    fiber.StatusOK,
			Status:  "success",
			Message: "Customer retrieved successfully",
			Data:    customer,
		})
}

func (ctrl *CustomerController) CreateCustomer(c *fiber.Ctx) error {
	req := new(validation.CreateCustomer)
	if err := c.BodyParser(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	customer, err := ctrl.CustomerService.CreateCustomer(c, req)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).
		JSON(response.SuccessWithData[*model.Customer]{
			Code:    fiber.StatusCreated,
			Status:  "success",
			Message: "Customer created successfully",
			Data:    customer,
		})
}

func (ctrl *CustomerController) UpdateCustomer(c *fiber.Ctx) error {
	id := c.Params("id")
	req := new(validation.UpdateCustomer)
	if err := c.BodyParser(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	customer, err := ctrl.CustomerService.UpdateCustomer(c, id, req)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).
		JSON(response.SuccessWithData[*model.Customer]{
			Code:    fiber.StatusOK,
			Status:  "success",
			Message: "Customer updated successfully",
			Data:    customer,
		})
}

func (ctrl *CustomerController) DeleteCustomer(c *fiber.Ctx) error {
	id := c.Params("id")
	err := ctrl.CustomerService.DeleteCustomer(c, id)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).
		JSON(response.Common{
			Code:    fiber.StatusOK,
			Status:  "success",
			Message: "Customer deleted successfully",
		})
}
