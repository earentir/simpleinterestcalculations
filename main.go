package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

// Product represents a financial product (card)
type Product struct {
	Name        string  `json:"name"`
	AnnualRate  float64 `json:"annualRate"`
	MonthlyCost float64 `json:"monthlyCost"`
}

// MonthlyData represents the data for each month
type MonthlyData struct {
	Year               int     `json:"year"`
	Month              string  `json:"month"`
	ActualInterest     float64 `json:"actualInterest"`
	CurrentProductName string  `json:"currentProductName"`
}

// MonthlyProducts represents the products available in a given month
type MonthlyProducts struct {
	Year     int       `json:"year"`
	Month    string    `json:"month"`
	Products []Product `json:"products"`
}

// Global flag variables
var (
	jsonFileName     string
	productsFileName string
	csvOutput        bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "interest",
		Short: "A tool to compare financial products and interests.",
		Run: func(cmd *cobra.Command, args []string) {
			monthlyData, err := loadDataFromJSON(jsonFileName)
			if err != nil {
				log.Fatalf("Error loading JSON data: %v", err)
			}

			productsData, err := loadProductsFromJSON(productsFileName)
			if err != nil {
				log.Fatalf("Error loading products data: %v", err)
			}

			printInterestComparisonTable(monthlyData, productsData, csvOutput)
			printProductComparisonTable(monthlyData, productsData, csvOutput)
			printFutureProductComparisons(monthlyData, productsData, csvOutput)
		},
	}

	// Define flags using Cobra
	rootCmd.PersistentFlags().StringVar(&jsonFileName, "jsondata", "interest_data.json", "path to JSON data file")
	rootCmd.PersistentFlags().StringVar(&productsFileName, "productsdata", "products_data.json", "path to products JSON file")
	rootCmd.PersistentFlags().BoolVar(&csvOutput, "csv", false, "output tables in CSV format")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func loadDataFromJSON(filename string) ([]MonthlyData, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var monthlyData []MonthlyData
	err = json.Unmarshal(data, &monthlyData)
	return monthlyData, err
}

func loadProductsFromJSON(filename string) ([]MonthlyProducts, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var productsData []MonthlyProducts
	err = json.Unmarshal(data, &productsData)
	return productsData, err
}

// formatHeader inserts newlines between words for table headers
func formatHeader(header string) string {
	return strings.ReplaceAll(header, " ", "\n")
}

func printInterestComparisonTable(monthlyData []MonthlyData, productsData []MonthlyProducts, csvOutput bool) {
	headers := []string{
		"Year",
		"Month",
		"Plan Rate",
		"Current Plan Cost",
		"Actual Interest",
		"Interest After Costs",
		"Estimated Interest",
		"Interest Diff.",
		"Capital",
		"Capital Diff.",
		"Estimated Deposit",
	}

	var table *tablewriter.Table
	var writer *csv.Writer

	if csvOutput {
		writer = csv.NewWriter(os.Stdout)
		defer writer.Flush()

		// Write headers
		if err := writer.Write(headers); err != nil {
			log.Fatalln("error writing CSV headers:", err)
		}
	} else {
		table = tablewriter.NewWriter(os.Stdout)
		// Format headers with newlines for compactness
		formattedHeaders := make([]string, len(headers))
		for i, header := range headers {
			formattedHeaders[i] = formatHeader(header)
		}
		table.SetHeader(formattedHeaders)

		// Set column alignment
		table.SetColumnAlignment([]int{
			tablewriter.ALIGN_DEFAULT, // Year
			tablewriter.ALIGN_DEFAULT, // Month
			tablewriter.ALIGN_RIGHT,   // Plan Rate
			tablewriter.ALIGN_RIGHT,   // Current Plan Cost
			tablewriter.ALIGN_RIGHT,   // Actual Interest
			tablewriter.ALIGN_RIGHT,   // Interest After Costs
			tablewriter.ALIGN_RIGHT,   // Estimated Interest
			tablewriter.ALIGN_RIGHT,   // Interest Diff.
			tablewriter.ALIGN_RIGHT,   // Capital
			tablewriter.ALIGN_RIGHT,   // Capital Diff.
			tablewriter.ALIGN_RIGHT,   // Estimated Deposit
		})
	}

	// Create a map from year-month to products for quick lookup
	productsMap := make(map[string][]Product)
	for _, mp := range productsData {
		key := fmt.Sprintf("%d-%s", mp.Year, mp.Month)
		productsMap[key] = mp.Products
	}

	var previousCapital float64
	for idx, data := range monthlyData {
		key := fmt.Sprintf("%d-%s", data.Year, data.Month)
		products, ok := productsMap[key]
		if !ok {
			fmt.Printf("Products not found for %s %d\n", data.Month, data.Year)
			continue
		}

		currentProductName := data.CurrentProductName
		var currentRate float64
		var currentMonthlyCost float64
		for _, product := range products {
			if product.Name == currentProductName {
				currentRate = product.AnnualRate
				currentMonthlyCost = product.MonthlyCost
				break
			}
		}

		if currentRate == 0 {
			fmt.Printf("Current product not found for %s %d\n", data.Month, data.Year)
			continue
		}

		// Compute Interest After Plan Cost
		interestAfterPlanCost := data.ActualInterest - currentMonthlyCost

		// Compute Capital
		capital := data.ActualInterest / (currentRate / 100 / 12)

		var estimatedInterest, interestDifference, diffCapital, estimatedDeposit float64
		if idx > 0 {
			// Calculate estimated interest based on previous capital
			estimatedInterest = previousCapital * (currentRate / 100 / 12)
			// Calculate difference between actual interest and estimated interest
			interestDifference = data.ActualInterest - estimatedInterest
			// Difference in capital
			diffCapital = capital - previousCapital
			// Estimated deposit (difference in capital minus actual interest)
			estimatedDeposit = diffCapital - data.ActualInterest
		} else {
			// For the first month, no previous capital
			estimatedInterest = 0
			interestDifference = 0
			diffCapital = 0
			estimatedDeposit = 0
		}

		// Update previousCapital for next iteration
		previousCapital = capital

		row := []string{
			fmt.Sprintf("%d", data.Year),
			data.Month,
			fmt.Sprintf("%.2f%%", currentRate),
			fmt.Sprintf("%.2f", currentMonthlyCost),
			fmt.Sprintf("%.2f", data.ActualInterest),
			fmt.Sprintf("%.2f", interestAfterPlanCost),
			fmt.Sprintf("%.2f", estimatedInterest),
			fmt.Sprintf("%.2f", interestDifference),
			fmt.Sprintf("%.2f", capital),
			fmt.Sprintf("%.2f", diffCapital),
			fmt.Sprintf("%.2f", estimatedDeposit),
		}

		if csvOutput {
			if err := writer.Write(row); err != nil {
				log.Fatalln("error writing CSV record:", err)
			}
		} else {
			// Create colors slice
			colors := make([]tablewriter.Colors, len(row))
			// For columns 2 onwards (indexes 2 to len(row)-1)
			for i := 2; i < len(row); i++ {
				// Parse the value
				valueStr := row[i]
				// Remove any percentage sign
				valueStr = strings.TrimSuffix(valueStr, "%")
				// Parse the value
				value, err := strconv.ParseFloat(valueStr, 64)
				if err != nil {
					// If parsing fails, no color
					colors[i] = tablewriter.Colors{}
					continue
				}
				if value < 0 {
					// Negative value, color red
					colors[i] = tablewriter.Colors{tablewriter.FgRedColor}
				} else if value > 0 && value < 1 {
					// Positive but less than 1, color yellow
					colors[i] = tablewriter.Colors{tablewriter.FgYellowColor}
				} else {
					// No color
					colors[i] = tablewriter.Colors{}
				}
			}
			// For Year and Month columns, no color
			colors[0] = tablewriter.Colors{}
			colors[1] = tablewriter.Colors{}

			// Add the row with colors
			table.Rich(row, colors)
		}
	}

	if !csvOutput {
		fmt.Println("Interest Comparison Table:")
		table.Render()
	}
}

func printProductComparisonTable(monthlyData []MonthlyData, productsData []MonthlyProducts, csvOutput bool) {
	// Get the last month's data
	lastMonthData := monthlyData[len(monthlyData)-1]
	year := lastMonthData.Year
	month := lastMonthData.Month

	// Get the products for the last month
	var products []Product
	for _, mp := range productsData {
		if mp.Year == year && mp.Month == month {
			products = mp.Products
			break
		}
	}

	if len(products) == 0 {
		fmt.Printf("Products not found for %s %d\n", month, year)
		return
	}

	// Use the current product name from the last month's data
	currentProductName := lastMonthData.CurrentProductName
	var currentRate float64
	for _, product := range products {
		if product.Name == currentProductName {
			currentRate = product.AnnualRate
			break
		}
	}

	if currentRate == 0 {
		fmt.Printf("Current product not found in the last month (%s %d)\n", month, year)
		return
	}

	// Compute the capital for the last month
	capital := lastMonthData.ActualInterest / (currentRate / 100 / 12)

	headers := []string{"Product", "Annual Rate", "Monthly Cost", "Projected Interest", "Net Gain"}

	var table *tablewriter.Table
	var writer *csv.Writer

	if csvOutput {
		writer = csv.NewWriter(os.Stdout)
		defer writer.Flush()

		// Write headers
		if err := writer.Write(headers); err != nil {
			log.Fatalln("error writing CSV headers:", err)
		}
	} else {
		table = tablewriter.NewWriter(os.Stdout)
		// Format headers with newlines for compactness
		formattedHeaders := make([]string, len(headers))
		for i, header := range headers {
			formattedHeaders[i] = formatHeader(header)
		}
		table.SetHeader(formattedHeaders)

		// Set column alignment
		table.SetColumnAlignment([]int{
			tablewriter.ALIGN_DEFAULT, // Product
			tablewriter.ALIGN_RIGHT,   // Annual Rate
			tablewriter.ALIGN_RIGHT,   // Monthly Cost
			tablewriter.ALIGN_RIGHT,   // Projected Interest
			tablewriter.ALIGN_RIGHT,   // Net Gain
		})
	}

	type ProductRow struct {
		Row         []string
		NetGain     float64
		ProductName string
	}

	var productRows []ProductRow
	var maxNetGain float64 = -1e9 // Initialize to a very small number

	// First pass: compute net gains and find the maximum net gain
	for _, product := range products {
		projectedInterest := capital * (product.AnnualRate / 100 / 12)
		netGain := projectedInterest - product.MonthlyCost

		row := []string{
			product.Name,
			fmt.Sprintf("%.2f%%", product.AnnualRate),
			fmt.Sprintf("%.2f", product.MonthlyCost),
			fmt.Sprintf("%.2f", projectedInterest),
			fmt.Sprintf("%.2f", netGain),
		}

		productRows = append(productRows, ProductRow{
			Row:         row,
			NetGain:     netGain,
			ProductName: product.Name,
		})

		// Update maxNetGain
		if netGain > maxNetGain {
			maxNetGain = netGain
		}
	}

	// Second pass: output the rows with appropriate colors
	for _, pr := range productRows {
		row := pr.Row
		if csvOutput {
			if err := writer.Write(row); err != nil {
				log.Fatalln("error writing CSV record:", err)
			}
		} else {
			// Create colors slice
			colors := make([]tablewriter.Colors, len(row))
			// Color the Net Gain column (index 4)
			if pr.NetGain == maxNetGain {
				// Highest net gain, color green
				colors[4] = tablewriter.Colors{tablewriter.FgGreenColor}
			} else if pr.NetGain < 0 {
				// Negative net gain, color red
				colors[4] = tablewriter.Colors{tablewriter.FgRedColor}
			} else {
				colors[4] = tablewriter.Colors{}
			}

			// Color the entire row yellow if this is the current plan
			if pr.ProductName == currentProductName {
				// Set foreground color yellow for all cells in the row
				for i := 0; i < len(row); i++ {
					colors[i] = tablewriter.Colors{tablewriter.FgHiYellowColor}
				}
			}

			table.Rich(row, colors)
		}
	}

	if !csvOutput {
		fmt.Printf("\nProduct Comparison Table for %s %d:\n", month, year)
		table.Render()
	}
}

func printFutureProductComparisons(monthlyData []MonthlyData, productsData []MonthlyProducts, csvOutput bool) {
	// Get the last date from the monthly data
	lastData := monthlyData[len(monthlyData)-1]
	lastDateStr := fmt.Sprintf("%d-%s", lastData.Year, lastData.Month)
	lastDate, err := time.Parse("2006-January", lastDateStr)
	if err != nil {
		fmt.Printf("Error parsing date '%s': %v\n", lastDateStr, err)
		return
	}

	// Compute the capital from the last month
	// Get the current rate from the last month's product
	var currentRate float64
	for _, mp := range productsData {
		if mp.Year == lastData.Year && mp.Month == lastData.Month {
			for _, product := range mp.Products {
				if product.Name == lastData.CurrentProductName {
					currentRate = product.AnnualRate
					break
				}
			}
			break
		}
	}

	if currentRate == 0 {
		fmt.Printf("Current rate not found for last month (%s %d)\n", lastData.Month, lastData.Year)
		return
	}

	capital := lastData.ActualInterest / (currentRate / 100 / 12)

	// Collect future months' products
	type FutureProduct struct {
		Date     time.Time
		Products []Product
	}
	var futureProducts []FutureProduct
	for _, mp := range productsData {
		// Parse the date
		dateStr := fmt.Sprintf("%d-%s", mp.Year, mp.Month)
		date, err := time.Parse("2006-January", dateStr)
		if err != nil {
			fmt.Printf("Error parsing date '%s': %v\n", dateStr, err)
			continue
		}

		if date.After(lastDate) {
			futureProducts = append(futureProducts, FutureProduct{
				Date:     date,
				Products: mp.Products,
			})
		}
	}

	if len(futureProducts) == 0 {
		fmt.Println("No future products found.")
		return
	}

	// Sort future products by date
	sort.Slice(futureProducts, func(i, j int) bool {
		return futureProducts[i].Date.Before(futureProducts[j].Date)
	})

	// Get the current product name from the last month's data
	currentProductName := lastData.CurrentProductName

	// For each future month, print the product comparison table
	for _, fp := range futureProducts {
		year, month := fp.Date.Year(), fp.Date.Format("January")

		headers := []string{"Product", "Annual Rate", "Monthly Cost", "Projected Interest", "Net Gain"}

		var table *tablewriter.Table
		var writer *csv.Writer

		if csvOutput {
			writer = csv.NewWriter(os.Stdout)
			defer writer.Flush()

			// Write headers
			if err := writer.Write(headers); err != nil {
				log.Fatalln("error writing CSV headers:", err)
			}
		} else {
			table = tablewriter.NewWriter(os.Stdout)
			// Format headers with newlines for compactness
			formattedHeaders := make([]string, len(headers))
			for i, header := range headers {
				formattedHeaders[i] = formatHeader(header)
			}
			table.SetHeader(formattedHeaders)

			// Set column alignment
			table.SetColumnAlignment([]int{
				tablewriter.ALIGN_DEFAULT, // Product
				tablewriter.ALIGN_RIGHT,   // Annual Rate
				tablewriter.ALIGN_RIGHT,   // Monthly Cost
				tablewriter.ALIGN_RIGHT,   // Projected Interest
				tablewriter.ALIGN_RIGHT,   // Net Gain
			})
		}

		type ProductRow struct {
			Row         []string
			NetGain     float64
			ProductName string
		}

		var productRows []ProductRow
		var maxNetGain float64 = -1e9 // Initialize to a very small number

		// First pass: compute net gains and find the maximum net gain
		for _, product := range fp.Products {
			projectedInterest := capital * (product.AnnualRate / 100 / 12)
			netGain := projectedInterest - product.MonthlyCost

			row := []string{
				product.Name,
				fmt.Sprintf("%.2f%%", product.AnnualRate),
				fmt.Sprintf("%.2f", product.MonthlyCost),
				fmt.Sprintf("%.2f", projectedInterest),
				fmt.Sprintf("%.2f", netGain),
			}

			productRows = append(productRows, ProductRow{
				Row:         row,
				NetGain:     netGain,
				ProductName: product.Name,
			})

			// Update maxNetGain
			if netGain > maxNetGain {
				maxNetGain = netGain
			}
		}

		// Second pass: output the rows with appropriate colors
		for _, pr := range productRows {
			row := pr.Row
			if csvOutput {
				if err := writer.Write(row); err != nil {
					log.Fatalln("error writing CSV record:", err)
				}
			} else {
				// Create colors slice
				colors := make([]tablewriter.Colors, len(row))
				// Color the Net Gain column (index 4)
				if pr.NetGain == maxNetGain {
					// Highest net gain, color green
					colors[4] = tablewriter.Colors{tablewriter.FgGreenColor}
				} else if pr.NetGain < 0 {
					// Negative net gain, color red
					colors[4] = tablewriter.Colors{tablewriter.FgRedColor}
				} else {
					colors[4] = tablewriter.Colors{}
				}

				// Color the entire row yellow if this is the current plan
				if pr.ProductName == currentProductName {
					// Set foreground color yellow for all cells in the row
					for i := 0; i < len(row); i++ {
						colors[i] = tablewriter.Colors{tablewriter.FgHiYellowColor}
					}
				}

				table.Rich(row, colors)
			}
		}

		if !csvOutput {
			fmt.Printf("\nProduct Comparison Table for %s %d:\n", month, year)
			table.Render()
		}
	}
}
