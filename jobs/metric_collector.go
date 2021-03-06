package jobs

import (
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/utils"
)

type metricCollector struct{}

func (m metricCollector) Name() string {
	return "metricCollector"
}

func (m metricCollector) ScheduleInterval() string {
	return "@every 1h"
}

func (m metricCollector) Run() {
	utils.SlackLog("running " + m.Name())

	m.spaceUsageMetrics()
	m.accountsMetrics()
	m.fileMetrics()
}

func (m metricCollector) Runnable() bool {
	return models.DB != nil
}

func (m metricCollector) spaceUsageMetrics() {
	spaceReport := models.CreateSpaceUsedReport()

	utils.Metrics_Percent_Of_Space_Used_Map[utils.TotalLbl].Set(models.CalculatePercentSpaceUsed(spaceReport))

	for _, plan := range utils.Env.Plans {
		spaceReport := models.CreateSpaceUsedReportForPlanType(models.StorageLimitType(plan.StorageInGB))

		utils.Metrics_Percent_Of_Space_Used_Map[plan.Name].Set(models.CalculatePercentSpaceUsed(spaceReport))
	}
}

func (m metricCollector) accountsMetrics() {
	accountsCount := 0
	totalAccountErr := models.DB.Model(&models.Account{}).Count(&accountsCount).Error
	if totalAccountErr == nil {
		utils.Metrics_Total_Accounts.Set(float64(accountsCount))
	}

	accountsPaidWithStripe := 0
	totalAccountPaidWithStripeErr := models.DB.Model(&models.Account{}).Where("payment_method = ?", models.PaymentMethodWithCreditCard).Count(&accountsPaidWithStripe).Error
	if totalAccountPaidWithStripeErr == nil {
		utils.Metrics_Total_Stripe_Paid_Accounts_Map[utils.TotalLbl].Set(float64(accountsPaidWithStripe))
	}

	unpaidAccountsCount, unpaidCountErr := models.CountAccountsByPaymentStatus(models.InitialPaymentInProgress)
	if unpaidCountErr == nil {
		utils.Metrics_Total_Unpaid_Accounts.Set(float64(unpaidAccountsCount))
	}

	collectedAccountsCount, collectedAccountErr := models.CountAccountsByPaymentStatus(models.PaymentRetrievalComplete)
	if collectedAccountErr == nil {
		utils.Metrics_Total_Collected_Accounts.Set(float64(collectedAccountsCount))
	}

	expiredAccountsCount, expiredAccountsErr := models.CountOfExpiredAccounts()
	if expiredAccountsErr == nil {
		utils.Metrics_Total_Expired_Accounts.Set(float64(expiredAccountsCount))
	}

	expiredArchivedAccountsCount, expiredArchivedAccountsErr := models.CountOfExpiredArchivedAccounts()
	if expiredArchivedAccountsErr == nil {
		utils.Metrics_Total_ExpiredArchived_Accounts.Set(float64(expiredArchivedAccountsCount))
	}

	renewedAccountsCount, renewedAccountsErr := models.CountOfRenewedAccounts()
	if renewedAccountsErr == nil {
		utils.Metrics_Total_Renewed_Accounts.Set(float64(renewedAccountsCount))
	}

	if utils.ReturnFirstError([]error{totalAccountErr, unpaidCountErr, collectedAccountErr}) == nil {
		paidAccountsCount := accountsCount - unpaidAccountsCount
		utils.Metrics_Total_Paid_Accounts_Map[utils.TotalLbl].Set(float64(paidAccountsCount))

		percentOfAccountsPaid := (float64(paidAccountsCount) / float64(accountsCount)) * float64(100)

		utils.Metrics_Percent_Of_Accounts_Paid.Set(float64(percentOfAccountsPaid))

		percentOfAccountsUnpaid := (float64(unpaidAccountsCount) / float64(accountsCount)) * float64(100)

		utils.Metrics_Percent_Of_Accounts_Unpaid.Set(float64(percentOfAccountsUnpaid))

		percentOfPaidAccountsCollected := (float64(collectedAccountsCount) / float64(paidAccountsCount)) * float64(100)

		utils.Metrics_Percent_Of_Paid_Accounts_Collected.Set(float64(percentOfPaidAccountsCollected))
	}

	for _, plan := range utils.Env.Plans {
		name := plan.Name
		accountCount, err := models.CountPaidAccountsByPlanType(models.StorageLimitType(plan.StorageInGB))
		if err == nil {
			utils.Metrics_Total_Paid_Accounts_Map[name].Set(float64(accountCount))
		}
		utils.LogIfError(err, nil)

		stripeCount, err := models.CountPaidAccountsByPaymentMethodAndPlanType(models.StorageLimitType(plan.StorageInGB), models.PaymentMethodWithCreditCard)
		if err == nil {
			utils.Metrics_Total_Stripe_Paid_Accounts_Map[name].Set(float64(stripeCount))
		}
		utils.LogIfError(err, nil)
	}
}

func (m metricCollector) fileMetrics() {
	completedFileInSQLCount := 0
	err := models.DB.Model(&models.CompletedFile{}).Count(&completedFileInSQLCount).Error
	utils.LogIfError(err, nil)
	if err == nil {
		utils.Metrics_Completed_Files_Count_SQL.Set(float64(completedFileInSQLCount))
	}

	fileSizeInByteInSQL, err := models.GetTotalFileSizeInByte()
	utils.LogIfError(err, nil)
	if err == nil {
		utils.Metrics_Uploaded_File_Size_MB_SQL.Set(float64(fileSizeInByteInSQL) / 1000000.0)
	}
}
