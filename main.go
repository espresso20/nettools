import (
	// "bufio"
	"fmt"
	"net"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/olekukonko/tablewriter"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage:", os.Args[0], "[-d duration_in_seconds] <ip_or_dns_name>")
		os.Exit(1)
	}

	var floatTimes []float64

	duration := 60
	var ipOrDNS string
	if os.Args[1] == "-d" && len(os.Args) > 3 {
		tempDuration, err := strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Println("Invalid duration specified:", os.Args[2])
			os.Exit(1)
		}
		duration = tempDuration
		ipOrDNS = os.Args[3]
	} else {
		ipOrDNS = os.Args[1]
	}

	fmt.Printf("The ping will run for %d seconds...\n", duration)

	cmd := exec.Command("ping", "-c", strconv.Itoa(duration), "-i", "1", ipOrDNS)
	output, _ := cmd.CombinedOutput()

	r := regexp.MustCompile(`time=([\d\.]+) ms`)
	matches := r.FindAllStringSubmatch(string(output), -1)

	printedFailMsg := false

	if matches == nil {
		fmt.Println("Failed to ping", ipOrDNS, "or no response in", duration, "seconds.")
		printedFailMsg = true
	} else {
		for _, match := range matches {
			time, _ := strconv.ParseFloat(match[1], 64)
			floatTimes = append(floatTimes, time)
		}
	}

	for _, match := range matches {
		time, _ := strconv.ParseFloat(match[1], 64)
		floatTimes = append(floatTimes, time)
	}

	// Check if provided argument is not an IP
	if net.ParseIP(ipOrDNS) == nil {
		ips, err := lookupIP(ipOrDNS)
		if err != nil {
			fmt.Printf("Failed to lookup IP for domain %s: %s\n", ipOrDNS, err)
		} else if len(ips) > 0 {
			fmt.Printf("\nDomain: %s\n", ipOrDNS)
			for _, ip := range ips {
				fmt.Printf("Resolved IP: %s\n", ip.String())
			}
			fmt.Println() // For an empty line before the next section
		}
	}

	// If no successful pings
	if len(floatTimes) == 0 && !printedFailMsg {
		fmt.Printf("Failed to ping %s or no response in %d seconds.\n", ipOrDNS, duration)
	} else if len(floatTimes) > 0 {
		drawTable(floatTimes, ipOrDNS, duration)
	}

	// Define popular ports to check
	ports := []int{22, 443, 80, 5432}
	portStatus := make(map[int]string)

	for _, port := range ports {
		if isPortOpen(ipOrDNS, port) {
			portStatus[port] = "Open/Responded"
		} else {
			portStatus[port] = "No Response"
		}
	}

	// Display port status in a table
	portTable := tablewriter.NewWriter(os.Stdout)
	portTable.SetHeader([]string{"Port", "Status"})

	for _, port := range ports {
		portTable.Append([]string{strconv.Itoa(port), portStatus[port]})
	}

	fmt.Println("Port Check Results:")
	portTable.Render()
	fmt.Println() // For an empty line before the next section

	// tcpConnTime, err := measureTCPConnTime(ipOrDNS, 80)
	// if err != nil {
	// 	fmt.Printf("\nUnable to connect to %s on port 80 (TCP): %v\n\n", ipOrDNS, err)
	// } else {
	// 	fmt.Printf("\nTime taken to connect to %s on port 80 (TCP): %s\n\n", ipOrDNS, tcpConnTime)
	// }

	fmt.Println("TCP Connection Time Results (Latency with 5 Second Count):")
	connTimeTable := tablewriter.NewWriter(os.Stdout)
	connTimeTable.SetHeader([]string{"Port", "Latency (ms)"})

	for _, port := range ports {
		connTime, err := measureTCPConnTime(ipOrDNS, port)
		if err != nil {
			connTimeTable.Append([]string{strconv.Itoa(port), "Unable to connect"})
		} else {
			connTimeTable.Append([]string{strconv.Itoa(port), fmt.Sprintf("%.2f ms", connTime.Seconds()*1000)})

		}
	}

	connTimeTable.Render()
	fmt.Println()

	// drawTable(floatTimes, ipOrDNS, duration)
}

func measureTCPConnTime(host string, port int) (time.Duration, error) {
	start := time.Now()
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), 5*time.Second)
	if err != nil {
		return 0, err
	}
	conn.Close()
	return time.Since(start), nil
}

func isPortOpen(host string, port int) bool {
	address := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.DialTimeout("tcp", address, 2*time.Second)
	if err != nil {
		return false
	}
	defer conn.Close()
	return true
}

func lookupIP(domain string) ([]net.IP, error) {
	ips, err := net.LookupIP(domain)
	if err != nil {
		return nil, err
	}
	return ips, nil
}

func drawTable(floatTimes []float64, ipOrDNS string, duration int) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Ping number", "Response Time (ms)", "Graph"})

	// Colors for alternating graph bars
	graphColors := []int{
		tablewriter.FgBlueColor,
		tablewriter.FgGreenColor,
	}

	for i, time := range floatTimes {
		bar := strings.Repeat("█", int(time/10))
		table.Rich([]string{strconv.Itoa(i + 1), fmt.Sprintf("%.2f ms", time), bar}, []tablewriter.Colors{{}, {}, {tablewriter.Normal, graphColors[i%2]}})
	}

	table.Render()

	var total float64
	for _, time := range floatTimes {
		total += time
	}
	average := total / float64(len(floatTimes))

	fmt.Println("Average response time for", ipOrDNS, "over", duration, "seconds:", average, "ms")
	if average > 200 {
		fmt.Println("This response time seems high.")
	}
}
