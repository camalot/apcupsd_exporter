package apcupsdexporter

import (
	"log"
	"strings"
	"time"

	"github.com/mdlayher/apcupsd"
	"github.com/prometheus/client_golang/prometheus"
)

var _ StatusSource = &apcupsd.Client{}

// A StatusSource is a type which can retrieve UPS status information from
// apcupsd.  It is implemented by *apcupsd.Client.
type StatusSource interface {
	Status() (*apcupsd.Status, error)
}

// A UPSCollector is a Prometheus collector for metrics regarding an APC UPS.
type UPSCollector struct {
	Info *prometheus.Desc

	UPSLoadPercent                      *prometheus.Desc
	BatteryChargePercent                *prometheus.Desc
	LineVolts                           *prometheus.Desc
	LineNominalVolts                    *prometheus.Desc
	OutputVolts                         *prometheus.Desc
	BatteryVolts                        *prometheus.Desc
	BatteryNominalVolts                 *prometheus.Desc
	BatteryNumberTransfersTotal         *prometheus.Desc
	BatteryTimeLeftSeconds              *prometheus.Desc
	BatteryTimeOnSeconds                *prometheus.Desc
	BatteryCumulativeTimeOnSecondsTotal *prometheus.Desc
	LastTransferOnBatteryTimeSeconds    *prometheus.Desc
	LastTransferOffBatteryTimeSeconds   *prometheus.Desc
	LastSelftestTimeSeconds             *prometheus.Desc
	NominalPowerWatts                   *prometheus.Desc
	Status                              *prometheus.Desc
	InternalTemperatureCelsius          *prometheus.Desc

	ss StatusSource
}

var _ prometheus.Collector = &UPSCollector{}

// NewUPSCollector creates a new UPSCollector.
func NewUPSCollector(ss StatusSource) *UPSCollector {
	labels := []string{"ups_name", "hostname", "model"}

	return &UPSCollector{
		Info: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "info"),
			"Metadata about a given UPS.",
			[]string{"ups_name", "hostname", "model"},
			nil,
		),

		Status: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "status"),
			"Current UPS status.",
			[]string{"ups_name", "hostname", "model", "status"},
			nil,
		),

		UPSLoadPercent: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "ups_load_percent"),
			"Current UPS load percentage.",
			labels,
			nil,
		),

		BatteryChargePercent: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "battery_charge_percent"),
			"Current UPS battery charge percentage.",
			labels,
			nil,
		),

		LineVolts: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "line_volts"),
			"Current AC input line voltage.",
			labels,
			nil,
		),

		LineNominalVolts: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "line_nominal_volts"),
			"Nominal AC input line voltage.",
			labels,
			nil,
		),

		OutputVolts: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "output_volts"),
			"Current AC output voltage.",
			labels,
			nil,
		),

		BatteryVolts: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "battery_volts"),
			"Current UPS battery voltage.",
			labels,
			nil,
		),

		BatteryNominalVolts: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "battery_nominal_volts"),
			"Nominal UPS battery voltage.",
			labels,
			nil,
		),

		BatteryNumberTransfersTotal: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "battery_number_transfers_total"),
			"Total number of transfers to UPS battery power.",
			labels,
			nil,
		),

		BatteryTimeLeftSeconds: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "battery_time_left_seconds"),
			"Number of seconds remaining of UPS battery power.",
			labels,
			nil,
		),

		BatteryTimeOnSeconds: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "battery_time_on_seconds"),
			"Number of seconds the UPS has been providing battery power due to an AC input line outage.",
			labels,
			nil,
		),

		BatteryCumulativeTimeOnSecondsTotal: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "battery_cumulative_time_on_seconds_total"),
			"Total number of seconds the UPS has provided battery power due to AC input line outages.",
			labels,
			nil,
		),

		LastTransferOnBatteryTimeSeconds: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "last_transfer_on_battery_time_seconds"),
			"UNIX timestamp of last transfer to battery since apcupsd startup.",
			labels,
			nil,
		),

		LastTransferOffBatteryTimeSeconds: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "last_transfer_off_battery_time_seconds"),
			"UNIX timestamp of last transfer from battery since apcupsd startup.",
			labels,
			nil,
		),

		LastSelftestTimeSeconds: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "last_selftest_time_seconds"),
			"UNIX timestamp of last selftest since apcupsd startup.",
			labels,
			nil,
		),

		NominalPowerWatts: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "nominal_power_watts"),
			"Nominal power output in watts.",
			labels,
			nil,
		),

		InternalTemperatureCelsius: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "internal_temperature_celsius"),
			"Internal temperature in °C.",
			labels,
			nil,
		),

		ss: ss,
	}
}

// Describe sends the descriptors of each metric over to the provided channel.
// The corresponding metric values are sent separately.
func (c *UPSCollector) Describe(ch chan<- *prometheus.Desc) {
	ds := []*prometheus.Desc{
		c.Info,
		c.Status,
		c.UPSLoadPercent,
		c.BatteryChargePercent,
		c.LineVolts,
		c.LineNominalVolts,
		c.OutputVolts,
		c.BatteryVolts,
		c.BatteryNominalVolts,
		c.BatteryNumberTransfersTotal,
		c.BatteryTimeLeftSeconds,
		c.BatteryTimeOnSeconds,
		c.BatteryCumulativeTimeOnSecondsTotal,
		c.LastTransferOnBatteryTimeSeconds,
		c.LastTransferOffBatteryTimeSeconds,
		c.LastSelftestTimeSeconds,
		c.NominalPowerWatts,
		c.InternalTemperatureCelsius,
	}

	for _, d := range ds {
		ch <- d
	}
}

// Collect sends the metric values for each metric created by the UPSCollector
// to the provided prometheus Metric channel.
func (c *UPSCollector) Collect(ch chan<- prometheus.Metric) {
	s, err := c.ss.Status()
	if err != nil {
		log.Printf("failed collecting UPS metrics: %v", err)
		ch <- prometheus.NewInvalidMetric(c.Info, err)
		return
	}

	upsStatus := []string{
		"CAL",           // Calibration mode
		"TRIM",          // Smart trim active
		"BOOST",         // Smart boost active
		"ONLINE",        // UPS is online
		"ONBATT",        // UPS is on battery
		"OVERLOAD",      // UPS is overloaded
		"LOWBATT",       // UPS has a low battery
		"REPLACEBATT",   // UPS battery needs to be replaced
		"NOBATT",        // UPS has no battery
		"SLAVE",         // UPS is a slave
		"SLAVEDOWN",     // UPS is a slave and is down
		"COMMLOST",      // Communication has been lost
		"SHUTTING DOWN", // UPS is shutting down
	}

	for _, status := range upsStatus {
		value := float64(0)
		if strings.Contains(s.Status, status) {
			value = float64(1)
		}
		ch <- prometheus.MustNewConstMetric(
			c.Status,
			prometheus.GaugeValue,
			value,
			s.UPSName, s.Hostname, s.Model, status,
		)
	}

	ch <- prometheus.MustNewConstMetric(
		c.Info,
		prometheus.GaugeValue,
		1,
		s.UPSName, s.Hostname, s.Model,
	)

	ch <- prometheus.MustNewConstMetric(
		c.UPSLoadPercent,
		prometheus.GaugeValue,
		s.LoadPercent,
		s.UPSName, s.Hostname, s.Model,
	)

	ch <- prometheus.MustNewConstMetric(
		c.BatteryChargePercent,
		prometheus.GaugeValue,
		s.BatteryChargePercent,
		s.UPSName, s.Hostname, s.Model,
	)

	ch <- prometheus.MustNewConstMetric(
		c.LineVolts,
		prometheus.GaugeValue,
		s.LineVoltage,
		s.UPSName, s.Hostname, s.Model,
	)

	ch <- prometheus.MustNewConstMetric(
		c.LineNominalVolts,
		prometheus.GaugeValue,
		s.NominalInputVoltage,
		s.UPSName, s.Hostname, s.Model,
	)

	ch <- prometheus.MustNewConstMetric(
		c.OutputVolts,
		prometheus.GaugeValue,
		s.OutputVoltage,
		s.UPSName, s.Hostname, s.Model,
	)

	ch <- prometheus.MustNewConstMetric(
		c.BatteryVolts,
		prometheus.GaugeValue,
		s.BatteryVoltage,
		s.UPSName, s.Hostname, s.Model,
	)

	ch <- prometheus.MustNewConstMetric(
		c.BatteryNominalVolts,
		prometheus.GaugeValue,
		s.NominalBatteryVoltage,
		s.UPSName, s.Hostname, s.Model,
	)

	ch <- prometheus.MustNewConstMetric(
		c.BatteryNumberTransfersTotal,
		prometheus.CounterValue,
		float64(s.NumberTransfers),
		s.UPSName, s.Hostname, s.Model,
	)

	ch <- prometheus.MustNewConstMetric(
		c.BatteryTimeLeftSeconds,
		prometheus.GaugeValue,
		s.TimeLeft.Seconds(),
		s.UPSName, s.Hostname, s.Model,
	)

	ch <- prometheus.MustNewConstMetric(
		c.BatteryTimeOnSeconds,
		prometheus.GaugeValue,
		s.TimeOnBattery.Seconds(),
		s.UPSName, s.Hostname, s.Model,
	)

	ch <- prometheus.MustNewConstMetric(
		c.BatteryCumulativeTimeOnSecondsTotal,
		prometheus.CounterValue,
		s.CumulativeTimeOnBattery.Seconds(),
		s.UPSName, s.Hostname, s.Model,
	)

	ch <- prometheus.MustNewConstMetric(
		c.LastTransferOnBatteryTimeSeconds,
		prometheus.GaugeValue,
		timestamp(s.XOnBattery),
		s.UPSName, s.Hostname, s.Model,
	)

	ch <- prometheus.MustNewConstMetric(
		c.LastTransferOffBatteryTimeSeconds,
		prometheus.GaugeValue,
		timestamp(s.XOffBattery),
		s.UPSName, s.Hostname, s.Model,
	)

	ch <- prometheus.MustNewConstMetric(
		c.LastSelftestTimeSeconds,
		prometheus.GaugeValue,
		timestamp(s.LastSelftest),
		s.UPSName, s.Hostname, s.Model,
	)

	ch <- prometheus.MustNewConstMetric(
		c.NominalPowerWatts,
		prometheus.GaugeValue,
		float64(s.NominalPower),
		s.UPSName, s.Hostname, s.Model,
	)

	ch <- prometheus.MustNewConstMetric(
		c.InternalTemperatureCelsius,
		prometheus.GaugeValue,
		s.InternalTemp,
		s.UPSName, s.Hostname, s.Model,
	)
}

func timestamp(t time.Time) float64 {
	if t.IsZero() {
		return 0
	}

	return float64(t.Unix())
}
