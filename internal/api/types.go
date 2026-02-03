package api

import "time"

// PaginationLinks contains links for paginated responses.
type PaginationLinks struct {
	First    string `json:"first"`
	Previous string `json:"previous"`
	Next     string `json:"next"`
	Last     string `json:"last"`
}

// Pagination contains common pagination fields.
type Pagination struct {
	PerPage      int             `json:"per_page"`
	TotalPages   int             `json:"total_pages"`
	TotalEntries int             `json:"total_entries"`
	NextPage     *int            `json:"next_page"`
	PreviousPage *int            `json:"previous_page"`
	Page         int             `json:"page"`
	Links        PaginationLinks `json:"links"`
}

// UserRef is a reference to a user in nested objects.
type UserRef struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// ClientRef is a reference to a client in nested objects.
type ClientRef struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// ProjectRef is a reference to a project in nested objects.
type ProjectRef struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Code string `json:"code,omitempty"`
}

// TaskRef is a reference to a task in nested objects.
type TaskRef struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// InvoiceRef is a reference to an invoice in nested objects.
type InvoiceRef struct {
	ID     int64  `json:"id"`
	Number string `json:"number"`
}

// ExternalReference contains external reference info for time entries.
type ExternalReference struct {
	ID             string `json:"id"`
	GroupID        string `json:"group_id"`
	AccountID      string `json:"account_id"`
	Permalink      string `json:"permalink"`
	Service        string `json:"service"`
	ServiceIconURL string `json:"service_icon_url"`
}

// UserAssignment represents a user's assignment to a project.
type UserAssignment struct {
	ID               int64     `json:"id"`
	IsProjectManager bool      `json:"is_project_manager"`
	IsActive         bool      `json:"is_active"`
	Budget           *float64  `json:"budget"`
	HourlyRate       *float64  `json:"hourly_rate"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// TaskAssignment represents a task's assignment to a project.
type TaskAssignment struct {
	ID         int64     `json:"id"`
	Billable   bool      `json:"billable"`
	IsActive   bool      `json:"is_active"`
	HourlyRate *float64  `json:"hourly_rate"`
	Budget     *float64  `json:"budget"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// TimeEntry represents a Harvest time entry.
type TimeEntry struct {
	ID                int64              `json:"id"`
	SpentDate         string             `json:"spent_date"`
	Hours             float64            `json:"hours"`
	HoursWithoutTimer float64            `json:"hours_without_timer"`
	RoundedHours      float64            `json:"rounded_hours"`
	Notes             string             `json:"notes"`
	IsLocked          bool               `json:"is_locked"`
	LockedReason      string             `json:"locked_reason"`
	IsClosed          bool               `json:"is_closed"`
	ApprovalStatus    string             `json:"approval_status"`
	IsBilled          bool               `json:"is_billed"`
	TimerStartedAt    *time.Time         `json:"timer_started_at"`
	StartedTime       string             `json:"started_time"`
	EndedTime         string             `json:"ended_time"`
	IsRunning         bool               `json:"is_running"`
	Billable          bool               `json:"billable"`
	Budgeted          bool               `json:"budgeted"`
	BillableRate      *float64           `json:"billable_rate"`
	CostRate          *float64           `json:"cost_rate"`
	User              UserRef            `json:"user"`
	Client            ClientRef          `json:"client"`
	Project           ProjectRef         `json:"project"`
	Task              TaskRef            `json:"task"`
	UserAssignment    *UserAssignment    `json:"user_assignment"`
	TaskAssignment    *TaskAssignment    `json:"task_assignment"`
	Invoice           *InvoiceRef        `json:"invoice"`
	ExternalReference *ExternalReference `json:"external_reference"`
	CreatedAt         time.Time          `json:"created_at"`
	UpdatedAt         time.Time          `json:"updated_at"`
}

// ProjectClientRef is the client reference within a project (includes currency).
type ProjectClientRef struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Currency string `json:"currency"`
}

// Project represents a Harvest project.
type Project struct {
	ID                               int64            `json:"id"`
	Name                             string           `json:"name"`
	Code                             string           `json:"code"`
	IsActive                         bool             `json:"is_active"`
	IsBillable                       bool             `json:"is_billable"`
	IsFixedFee                       bool             `json:"is_fixed_fee"`
	BillBy                           string           `json:"bill_by"`
	HourlyRate                       *float64         `json:"hourly_rate"`
	BudgetBy                         string           `json:"budget_by"`
	BudgetIsMonthly                  bool             `json:"budget_is_monthly"`
	Budget                           *float64         `json:"budget"`
	CostBudget                       *float64         `json:"cost_budget"`
	CostBudgetIncludeExpenses        bool             `json:"cost_budget_include_expenses"`
	NotifyWhenOverBudget             bool             `json:"notify_when_over_budget"`
	OverBudgetNotificationPercentage float64          `json:"over_budget_notification_percentage"`
	OverBudgetNotificationDate       *string          `json:"over_budget_notification_date"`
	ShowBudgetToAll                  bool             `json:"show_budget_to_all"`
	Fee                              *float64         `json:"fee"`
	Notes                            string           `json:"notes"`
	StartsOn                         *string          `json:"starts_on"`
	EndsOn                           *string          `json:"ends_on"`
	Client                           ProjectClientRef `json:"client"`
	CreatedAt                        time.Time        `json:"created_at"`
	UpdatedAt                        time.Time        `json:"updated_at"`
}

// Task represents a Harvest task.
type Task struct {
	ID                int64     `json:"id"`
	Name              string    `json:"name"`
	BillableByDefault bool      `json:"billable_by_default"`
	DefaultHourlyRate float64   `json:"default_hourly_rate"`
	IsDefault         bool      `json:"is_default"`
	IsActive          bool      `json:"is_active"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// User represents a Harvest user.
type User struct {
	ID                           int64     `json:"id"`
	FirstName                    string    `json:"first_name"`
	LastName                     string    `json:"last_name"`
	Email                        string    `json:"email"`
	Telephone                    string    `json:"telephone"`
	Timezone                     string    `json:"timezone"`
	HasAccessToAllFutureProjects bool      `json:"has_access_to_all_future_projects"`
	IsContractor                 bool      `json:"is_contractor"`
	IsActive                     bool      `json:"is_active"`
	WeeklyCapacity               int       `json:"weekly_capacity"`
	DefaultHourlyRate            *float64  `json:"default_hourly_rate"`
	CostRate                     *float64  `json:"cost_rate"`
	Roles                        []string  `json:"roles"`
	AccessRoles                  []string  `json:"access_roles"`
	AvatarURL                    string    `json:"avatar_url"`
	CreatedAt                    time.Time `json:"created_at"`
	UpdatedAt                    time.Time `json:"updated_at"`
}

// FullName returns the user's full name.
func (u *User) FullName() string {
	return u.FirstName + " " + u.LastName
}

// HarvestClient represents a Harvest client (customer).
type HarvestClient struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	IsActive     bool      `json:"is_active"`
	Address      string    `json:"address"`
	StatementKey string    `json:"statement_key"`
	Currency     string    `json:"currency"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Company represents a Harvest company/account.
type Company struct {
	BaseURI               string `json:"base_uri"`
	FullDomain            string `json:"full_domain"`
	Name                  string `json:"name"`
	IsActive              bool   `json:"is_active"`
	WeekStartDay          string `json:"week_start_day"`
	WantsTimestampTimers  bool   `json:"wants_timestamp_timers"`
	TimeFormat            string `json:"time_format"`
	DateFormat            string `json:"date_format"`
	PlanType              string `json:"plan_type"`
	Clock                 string `json:"clock"`
	CurrencyCodeDisplay   string `json:"currency_code_display"`
	CurrencySymbolDisplay string `json:"currency_symbol_display"`
	DecimalSymbol         string `json:"decimal_symbol"`
	ThousandsSeparator    string `json:"thousands_separator"`
	ColorScheme           string `json:"color_scheme"`
	WeeklyCapacity        int    `json:"weekly_capacity"`
	ExpenseFeature        bool   `json:"expense_feature"`
	InvoiceFeature        bool   `json:"invoice_feature"`
	EstimateFeature       bool   `json:"estimate_feature"`
	ApprovalFeature       bool   `json:"approval_feature"`
	TeamFeature           bool   `json:"team_feature"`
}

// TimeEntryInput is used to create or update a time entry.
type TimeEntryInput struct {
	UserID            *int64             `json:"user_id,omitempty"`
	ProjectID         int64              `json:"project_id,omitempty"`
	TaskID            int64              `json:"task_id,omitempty"`
	SpentDate         string             `json:"spent_date,omitempty"`
	Hours             *float64           `json:"hours,omitempty"`
	Notes             *string            `json:"notes,omitempty"`
	StartedTime       *string            `json:"started_time,omitempty"`
	EndedTime         *string            `json:"ended_time,omitempty"`
	ExternalReference *ExternalReference `json:"external_reference,omitempty"`
}

// ProjectInput is used to create or update a project.
type ProjectInput struct {
	ClientID                         int64    `json:"client_id,omitempty"`
	Name                             string   `json:"name,omitempty"`
	Code                             *string  `json:"code,omitempty"`
	IsActive                         *bool    `json:"is_active,omitempty"`
	IsBillable                       *bool    `json:"is_billable,omitempty"`
	IsFixedFee                       *bool    `json:"is_fixed_fee,omitempty"`
	BillBy                           string   `json:"bill_by,omitempty"`
	HourlyRate                       *float64 `json:"hourly_rate,omitempty"`
	BudgetBy                         string   `json:"budget_by,omitempty"`
	BudgetIsMonthly                  *bool    `json:"budget_is_monthly,omitempty"`
	Budget                           *float64 `json:"budget,omitempty"`
	CostBudget                       *float64 `json:"cost_budget,omitempty"`
	CostBudgetIncludeExpenses        *bool    `json:"cost_budget_include_expenses,omitempty"`
	NotifyWhenOverBudget             *bool    `json:"notify_when_over_budget,omitempty"`
	OverBudgetNotificationPercentage *float64 `json:"over_budget_notification_percentage,omitempty"`
	ShowBudgetToAll                  *bool    `json:"show_budget_to_all,omitempty"`
	Fee                              *float64 `json:"fee,omitempty"`
	Notes                            *string  `json:"notes,omitempty"`
	StartsOn                         *string  `json:"starts_on,omitempty"`
	EndsOn                           *string  `json:"ends_on,omitempty"`
}

// TaskInput is used to create or update a task.
type TaskInput struct {
	Name              string   `json:"name,omitempty"`
	BillableByDefault *bool    `json:"billable_by_default,omitempty"`
	DefaultHourlyRate *float64 `json:"default_hourly_rate,omitempty"`
	IsDefault         *bool    `json:"is_default,omitempty"`
	IsActive          *bool    `json:"is_active,omitempty"`
}

// ClientInput is used to create or update a client.
type ClientInput struct {
	Name     string  `json:"name,omitempty"`
	IsActive *bool   `json:"is_active,omitempty"`
	Address  *string `json:"address,omitempty"`
	Currency *string `json:"currency,omitempty"`
}

// UserInput is used to create or update a user.
type UserInput struct {
	FirstName                    string   `json:"first_name,omitempty"`
	LastName                     string   `json:"last_name,omitempty"`
	Email                        string   `json:"email,omitempty"`
	Timezone                     *string  `json:"timezone,omitempty"`
	HasAccessToAllFutureProjects *bool    `json:"has_access_to_all_future_projects,omitempty"`
	IsContractor                 *bool    `json:"is_contractor,omitempty"`
	IsActive                     *bool    `json:"is_active,omitempty"`
	WeeklyCapacity               *int     `json:"weekly_capacity,omitempty"`
	DefaultHourlyRate            *float64 `json:"default_hourly_rate,omitempty"`
	CostRate                     *float64 `json:"cost_rate,omitempty"`
	Roles                        []string `json:"roles,omitempty"`
	AccessRoles                  []string `json:"access_roles,omitempty"`
}

// ProjectAssignment represents a user's assignment to a project (from my/project_assignments).
type ProjectAssignment struct {
	ID               int64                   `json:"id"`
	IsProjectManager bool                    `json:"is_project_manager"`
	IsActive         bool                    `json:"is_active"`
	Budget           *float64                `json:"budget"`
	HourlyRate       *float64                `json:"hourly_rate"`
	CreatedAt        time.Time               `json:"created_at"`
	UpdatedAt        time.Time               `json:"updated_at"`
	Project          ProjectRef              `json:"project"`
	Client           ClientRef               `json:"client"`
	TaskAssignments  []ProjectTaskAssignment `json:"task_assignments"`
}

// ProjectTaskAssignment represents a task assignment within a project assignment.
type ProjectTaskAssignment struct {
	ID         int64    `json:"id"`
	Billable   bool     `json:"billable"`
	IsActive   bool     `json:"is_active"`
	HourlyRate *float64 `json:"hourly_rate"`
	Budget     *float64 `json:"budget"`
	Task       TaskRef  `json:"task"`
}

// ExpenseCategoryRef is a reference to an expense category in nested objects.
type ExpenseCategoryRef struct {
	ID        int64    `json:"id"`
	Name      string   `json:"name"`
	UnitPrice *float64 `json:"unit_price"`
	UnitName  *string  `json:"unit_name"`
}

// Receipt represents an expense receipt attachment.
type Receipt struct {
	URL         string `json:"url"`
	FileName    string `json:"file_name"`
	FileSize    int64  `json:"file_size"`
	ContentType string `json:"content_type"`
}

// Expense represents a Harvest expense.
type Expense struct {
	ID              int64              `json:"id"`
	Notes           string             `json:"notes"`
	TotalCost       float64            `json:"total_cost"`
	Units           float64            `json:"units"`
	IsClosed        bool               `json:"is_closed"`
	ApprovalStatus  string             `json:"approval_status"`
	IsLocked        bool               `json:"is_locked"`
	IsBilled        bool               `json:"is_billed"`
	LockedReason    string             `json:"locked_reason"`
	SpentDate       string             `json:"spent_date"`
	Billable        bool               `json:"billable"`
	Receipt         *Receipt           `json:"receipt"`
	User            UserRef            `json:"user"`
	UserAssignment  *UserAssignment    `json:"user_assignment"`
	Project         ProjectRef         `json:"project"`
	ExpenseCategory ExpenseCategoryRef `json:"expense_category"`
	Client          ClientRef          `json:"client"`
	Invoice         *InvoiceRef        `json:"invoice"`
	CreatedAt       time.Time          `json:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at"`
}

// ExpenseCategory represents a Harvest expense category.
type ExpenseCategory struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	UnitName  *string   `json:"unit_name"`
	UnitPrice *float64  `json:"unit_price"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ExpenseInput is used to create or update an expense.
type ExpenseInput struct {
	UserID            *int64   `json:"user_id,omitempty"`
	ProjectID         int64    `json:"project_id,omitempty"`
	ExpenseCategoryID int64    `json:"expense_category_id,omitempty"`
	SpentDate         string   `json:"spent_date,omitempty"`
	Units             *int     `json:"units,omitempty"`
	TotalCost         *float64 `json:"total_cost,omitempty"`
	Notes             *string  `json:"notes,omitempty"`
	Billable          *bool    `json:"billable,omitempty"`
	DeleteReceipt     *bool    `json:"delete_receipt,omitempty"`
}

// EstimateLineItem represents a line item on an estimate.
type EstimateLineItem struct {
	ID          int64   `json:"id,omitempty"`
	Kind        string  `json:"kind"`
	Description string  `json:"description,omitempty"`
	Quantity    float64 `json:"quantity,omitempty"`
	UnitPrice   float64 `json:"unit_price"`
	Amount      float64 `json:"amount,omitempty"`
	Taxed       bool    `json:"taxed,omitempty"`
	Taxed2      bool    `json:"taxed2,omitempty"`
	Destroy     bool    `json:"_destroy,omitempty"`
}

// Estimate represents a Harvest estimate.
type Estimate struct {
	ID             int64              `json:"id"`
	ClientKey      string             `json:"client_key"`
	Number         string             `json:"number"`
	PurchaseOrder  string             `json:"purchase_order"`
	Amount         float64            `json:"amount"`
	Tax            *float64           `json:"tax"`
	TaxAmount      float64            `json:"tax_amount"`
	Tax2           *float64           `json:"tax2"`
	Tax2Amount     float64            `json:"tax2_amount"`
	Discount       *float64           `json:"discount"`
	DiscountAmount float64            `json:"discount_amount"`
	Subject        string             `json:"subject"`
	Notes          string             `json:"notes"`
	Currency       string             `json:"currency"`
	State          string             `json:"state"`
	IssueDate      string             `json:"issue_date"`
	SentAt         *time.Time         `json:"sent_at"`
	AcceptedAt     *time.Time         `json:"accepted_at"`
	DeclinedAt     *time.Time         `json:"declined_at"`
	CreatedAt      time.Time          `json:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at"`
	Client         ClientRef          `json:"client"`
	Creator        UserRef            `json:"creator"`
	LineItems      []EstimateLineItem `json:"line_items"`
}

// EstimateInput is used to create or update an estimate.
type EstimateInput struct {
	ClientID      int64              `json:"client_id,omitempty"`
	Number        *string            `json:"number,omitempty"`
	PurchaseOrder *string            `json:"purchase_order,omitempty"`
	Tax           *float64           `json:"tax,omitempty"`
	Tax2          *float64           `json:"tax2,omitempty"`
	Discount      *float64           `json:"discount,omitempty"`
	Subject       *string            `json:"subject,omitempty"`
	Notes         *string            `json:"notes,omitempty"`
	Currency      *string            `json:"currency,omitempty"`
	IssueDate     *string            `json:"issue_date,omitempty"`
	LineItems     []EstimateLineItem `json:"line_items,omitempty"`
}

// EstimateMessageRecipient represents a recipient of an estimate message.
type EstimateMessageRecipient struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email"`
}

// EstimateMessage represents a message sent with an estimate.
type EstimateMessage struct {
	ID            int64                      `json:"id"`
	SentBy        string                     `json:"sent_by"`
	SentByEmail   string                     `json:"sent_by_email"`
	SentFrom      string                     `json:"sent_from"`
	SentFromEmail string                     `json:"sent_from_email"`
	Recipients    []EstimateMessageRecipient `json:"recipients"`
	Subject       string                     `json:"subject"`
	Body          string                     `json:"body"`
	SendMeACopy   bool                       `json:"send_me_a_copy"`
	EventType     string                     `json:"event_type"`
	CreatedAt     time.Time                  `json:"created_at"`
	UpdatedAt     time.Time                  `json:"updated_at"`
}

// EstimateMessageInput is used to create an estimate message.
type EstimateMessageInput struct {
	Recipients  []EstimateMessageRecipient `json:"recipients,omitempty"`
	Subject     *string                    `json:"subject,omitempty"`
	Body        *string                    `json:"body,omitempty"`
	SendMeACopy *bool                      `json:"send_me_a_copy,omitempty"`
	EventType   *string                    `json:"event_type,omitempty"`
}
