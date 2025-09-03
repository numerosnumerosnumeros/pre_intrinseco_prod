<script setup>
import { useHead } from '@unhead/vue'
import { config } from '@config'

useHead({
  title: 'nodo.finance - Documentation',
  meta: [
    {
      name: 'twitter:title',
      content: 'nodo.finance - Documentation',
    },
    {
      property: 'og:title',
      content: 'nodo.finance - Documentation',
    },
    {
      name: 'robots',
      content: 'index, follow',
    },
  ],
})

function scrollToElement(id) {
  const element = document.getElementById(id)
  if (element) {
    element.scrollIntoView({ behavior: 'smooth' })
  }
}
</script>

<template>
  <!-- eslint-disable -->
  <div class="wrapper">
    <h1>Documentation</h1>
    <p class="important">
      nodo.finance is currently in <b>beta</b> phase, so it may contain errors,
      occasional interruptions or frequent changes. Your <i>feedback</i> is
      valuable and contributes significantly to our development phases:
      <a :href="`mailto:${config.contactEmail}`">{{ config.contactEmail }}</a>
    </p>
    <p>
      To access nodo.finance's professional services you must be registered with
      a verified email and have an active professional plan. The email used
      during subscription must be the same one associated with your account. To
      cancel your subscription, click on '<i>Plan: Professional</i>' in your
      profile.
    </p>
    <p>Please, before subscribing review the terms of service:</p>
    <RouterLink to="/terms-of-service" class="nav-link">
      <p>Terms of service &#8594;</p>
    </RouterLink>
    <p class="comment">
      Periodically review the Documentation to stay up to date with new
      features.
    </p>
    <h2 id="index">Index</h2>
    <ul class="index">
      <li class="subindex">
        <a @click.prevent="scrollToElement('app-ratios')" class="scroll"
          >Ratios</a
        >
      </li>
      <li class="subindex">
        <a @click.prevent="scrollToElement('app-speculate')" class="scroll"
          >Speculate</a
        >
      </li>
      <li class="subindex">
        <a @click.prevent="scrollToElement('app-analyst')" class="scroll"
          >Analyst</a
        >
      </li>
      <li class="subindex">
        <a @click.prevent="scrollToElement('app-errors')" class="scroll"
          >Common errors</a
        >
      </li>
      <li class="subindex">
        <a @click.prevent="scrollToElement('app-limits')" class="scroll"
          >Limits</a
        >
      </li>
    </ul>

    <p>
      Our artificial intelligence will extract and analyze data from any
      financial report, no matter how extensive, in both <b>English</b> and
      <b>Spanish</b>.
    </p>
    <p>
      URLs and files in the following formats are valid: <b>.pdf</b>,
      <b>.html</b>, <b>.xhtml</b> and <b>.mht</b>, with a limit of
      <b>150MB</b> per file. When you want to attach a web page, you can do so
      by writing the URL selecting the 'url' option or through an .mht file (<i
        >right click → save as → single file</i
      >).
    </p>
    <p class="comment">
      Don't attach websites with files if it's the file you're interested in
      since no browser interactions are performed. In that case, download or
      navigate to the file you're interested in and attach it directly or send
      the URL.
    </p>
    <p class="comment">
      Client-side rendered web pages that require Javascript to load the HTML
      are not processed. If the website renders in the browser, download and use
      the .mht file as explained above. To check if the website is client-side
      rendered, disable Javascript in your browser and reload the page. If you
      don't see the content, you'll need to download the .mht file.
    </p>
    <p>
      The '<i>start</i>' and '<i>end</i>' fields are used to select the page
      range you want to process. By default it processes the first 100 pages of
      the file. It is recommended to fill in the values for the following cases:
    </p>
    <ul>
      <li>
        <b>Improve processing speed</b>. Huge files require longer processing
        time and consumption. If you're sure the results appear in the first 10
        pages for example, select 10 for '<i>end</i>'.
      </li>
      <li>
        <b>Cover more pages</b>. If you want to process more than 100 pages. For
        example, ASML company presentations usually contain financial reports
        around page 240. You should mark '<i>start</i>': 240 and if you want to
        be more precise, '<i>end</i>': 260. In this case, if no range is
        selected, the part containing the data will not be processed, since it's
        located very far from the beginning.
      </li>
      <li>
        <b>Increase precision</b>. For example, many CNMV reports contain both
        individual and consolidated financial reports, which can lead to errors.
        If you know that the consolidated ones appear from page 10 onwards, you
        can mark '<i>start</i>': 10.
      </li>
    </ul>
    <p>
      The date for '<i>period</i>' must be written in one of these 3 formats:
    </p>
    <ul>
      <li>
        <b>YYYY-Y</b> year, e.g.: 2024-Y. To prevent errors the -Y suffix is
        required.
      </li>
      <p class="important">
        It is recommended to analyze annual reports to avoid errors, since dates
        and periods are usually more specific and there is less possibility of
        using the wrong values.
      </p>
      <li>
        <b>YYYY-Q</b>: year and quarter, e.g.: 2024-Q1. In case '3 previous
        months' and '9 previous months' are reported, it will try to extract
        data relative to 3 previous months.
      </li>
      <li><b>YYYY-S</b> year and semester, e.g.: 2024-S1.</li>
    </ul>

    <p>
      The selected currency will be used to represent the financial report data,
      make sure it is correct. You can also select '-' to not use any specific
      currency.
    </p>
    <p class="important">
      If once you've added a company with a specific currency, you later add the
      same company for the same or another period with a different currency
      selected, the currency for that company will be overwritten.
    </p>

    <p>
      Currently our AI is limited to
      <b>{{ config.maxTickers }} companies</b> per user and
      <b>{{ config.maxPeriods }} periods per company</b>. The search for a
      company returns 20 results, use unique names for each company. If you
      repeat a period for a company, whose company-period combination has
      already been added, the new result will overwrite the previous one.
    </p>
    <p>
      Due to the size and complexity of the processed files, the organization
      and clarity of the content is crucial for correct extraction. The AI can
      make errors. Once a company is added, if you detect any error you can
      manually modify each value.
    </p>
    <p>
      You can delete specific periods. To delete a company simply delete all its
      periods.
    </p>
    <p>
      Once inside your company you will be able to: examine your company's
      finances, compare its data YoY (year-over-year), visualize its main
      margins with a Sankey chart, speculate with potential scenarios about
      desired price or P/E ratios, or request a financial analysis.
    </p>
    <p class="comment">
      Pay special attention if the company reports its results for the last
      period (quarterly or annual) in January of the following year. Certain
      companies name their reports in the year they present them. For example,
      MongoDB on 31/01/2024 presents its report containing information from
      01/02/2023 to 31/01/2024, practically all of 2023. In other cases, for
      example Inditex, it can be observed that in its 2022 report, it uses
      31/01/2023 for the balance sheet and 2022 for the income statement. To
      address these complex scenarios and guarantee consistency with the
      reflected data, the AI processes reports so that for MongoDB it would
      return the 2023 annual report and for Inditex the 2022 annual report,
      since those are the years covered. So if you were to analyze these
      companies, you would use '2023-Y' as the period for MongoDB and '2022-Y'
      for Inditex.
    </p>
    <p class="important">
      It is not recommended to use the AI to analyze companies from sectors that
      do not follow traditional formats for their financial statements, such as
      insurance companies or banks. In these cases, some of the data that the AI
      tries to obtain is not reported, which will lead to errors and
      inconsistencies.
    </p>
    <a @click.prevent="scrollToElement('index')" class="index-link"
      ><p>&#8593; index</p></a
    >

    <h3 id="app-ratios">Ratios</h3>
    <p class="comment">
      Abbreviations have been used for data nomenclature to simplify
      visualization. The data shown for each company is as follows:
    </p>
    <p>Data that requires a stock price to be calculated:</p>
    <ul>
      <li>
        <b>P / E</b>: <i>Price to Earnings Ratio</i>, stock price divided by net
        earnings per share.
      </li>
      <li>
        <b>P / BV</b>: <i>Price</i> / Book Value, stock price divided by book
        value per share.
      </li>
      <li>
        <b>EV</b>: <i>Enterprise Value</i>, market capitalization plus debt
        minus cash.
      </li>
      <li><b>EV cap</b>: <i>Enterprise Value</i> to market capitalization.</li>
      <li>
        <b>EV / OCF</b>: <i>Enterprise Value</i> divided by operating cash flow.
      </li>
      <li><b>EV / π</b>: <i>Enterprise Value</i> divided by net profit.</li>
    </ul>
    <p>Other data:</p>
    <ul>
      <li><b>CA</b>: Current Assets.</li>
      <li><b>NCA</b>: Non-Current Assets.</li>
      <li><b>TA</b>: Total Assets.</li>
      <li><b>Cash</b>: Cash and cash equivalents.</li>
      <li><b>CL</b>: Current Liabilities.</li>
      <li><b>NCL</b>: Non-Current Liabilities.</li>
      <li><b>TL</b>: Total Liabilities.</li>
      <li><b>SE</b>: Shareholders' Equity.</li>
      <li>
        <b>WC</b>: Working Capital, current assets minus current liabilities.
      </li>
      <li>
        <b>WC / NCL</b>: Working Capital divided by non-current liabilities.
      </li>
      <li>
        <b>Shares ≈</b>: approximate shares. Given the inconsistency in
        financial file formats and different accounting practices, an
        approximation is calculated using net profit and EPS.
      </li>
      <li><b>BV</b>: Book Value, shareholders' equity divided by shares.</li>
      <li><b>R</b>: Revenue.</li>
      <li><b>π</b>: Net Profit.</li>
      <li>
        <b>EPS</b>: Earnings Per Share. Will try to obtain the non-diluted
        value.
      </li>
      <li><b>Net margin</b>: Net profit divided by revenue.</li>
      <li>
        <b>ROA</b>: <i>Return on Assets</i>, net profit divided by total assets.
      </li>
      <li>
        <b>ROE</b>: <i>Return on Equity</i>, net profit divided by shareholders'
        equity.
      </li>
      <li><b>Liquidity</b>: current assets divided by current liabilities.</li>
      <li>
        <b>Solvency</b>: total assets divided by total liabilities. If
        shareholders' equity is negative, "NA" will be shown, not available.
      </li>
      <li>
        <b>Leverage</b>: total liabilities divided by shareholders' equity. If
        shareholders' equity is negative, "NA" will be shown, not available.
      </li>
      <li><b>Op CF</b>: operating cash flow.</li>
      <li><b>Inv CF</b>: investment cash flow.</li>
      <li><b>Fin CF</b>: financing cash flow.</li>
    </ul>
    <p>
      If you have added the counterpart period from the previous year, the
      <b>YoY</b> variation will be shown as a percentage next to each data
      point.
    </p>
    <a @click.prevent="scrollToElement('index')" class="index-link"
      ><p>&#8593; index</p></a
    >

    <h3 id="app-speculate">Speculate</h3>
    <p>
      Within each company, in the upper section there are 2 boxes where you can
      enter a value. In one of them the company's currency will appear and you
      can enter a price for each share of your company, for example the current
      price or the desired price. In the other one 'P / E' is shown and you can
      enter a desired P/E ratio.
    </p>
    <ul>
      <li>
        If you have filled in the price box, a score for the company at that
        specific price will be shown (see score explanation below).
      </li>
      <li>
        If you have filled in the P/E ratio box, the price needed to reach the
        desired P/E ratio will be shown.
      </li>
      <li>
        If you have filled in both boxes, the net profit required to reach the
        desired P/E ratio at that price and the percentage variation required
        compared to the current net profit will be shown.
      </li>
    </ul>
    <p>The formula for the score is as follows:</p>
    <pre>
Score = {
    NA,                    if Net Profit, Shares, EV, or P/BV are not valid
    0,                     if EPS ≤ 0 or BV ≤ 0 or Op CF ≤ 0 or Net Profit ≤ 0
    10,                    if the above are not met and EV ≤ 0
    W₁R₁ + W₂R₂ + W₃R₃,    otherwise
}</pre
    >
    <p>Where:</p>
    <pre>
R₁ = 10(1 - EV/OCF/50),    if EV/OCF < 50,     otherwise = 0
R₂ = 10(1 - P/E/50),       if P/E < 50,        otherwise = 0
R₃ = 10(1 - P/BV/20),      if P/BV < 20,       otherwise = 0</pre
    >
    <p>Weighting if all ratios are available:</p>
    <pre>
W₁ = 0.4 (weight of EV/OCF)
W₂ = 0.3 (weight of P/E)
W₃ = 0.3 (weight of P/BV)</pre
    >
    <p>Weighting if EV/OCF is not available:</p>
    <pre>
W₂ = 0.5 (weight of P/E)
W₃ = 0.5 (weight of P/BV)</pre
    >
    <p class="important">
      A negative variation for any of the data that requires a stock price
      implies that the company has improved in terms of valuation. For example,
      a lower P/E ratio or P/BV would mean a more attractive valuation given the
      constant price for both periods.
    </p>
    <p class="comment">
      Keep in mind that the score does not consider the different nature of each
      sector. It is useful for obtaining a quick and static view of the company,
      but each sector must be examined carefully. For example, a high P/E ratio
      of a fast-growing technology company may be more acceptable than that of a
      slow-growing utility company. In the mining sector, profits can be very
      volatile due to commodity cycles. Pharmaceutical companies may have
      prolonged periods of R&D investment before generating profits. The real
      estate sector is usually better valued by its net asset value than by its
      profits. Banks require specific analysis of asset quality and capital
      ratios. Software companies may have initial losses while building a
      customer base. Cyclical industrial companies should be evaluated over an
      entire economic cycle to have a more accurate picture.
    </p>
    <p>
      It is recommended to use the speculation functionalities with
      <b>annual</b>
      data. If you speculate with quarterly or semi-annual data, the values will
      not reflect the company's annual situation. For example, the net result
      will be from that quarter or semester, not from the full year, so the
      score will be worse and the required price and profit higher. An
      approximation could be to multiply/divide by 4 for quarterly and by 2 for
      semi-annual, although it will not reflect factors such as seasonality.
    </p>
    <a @click.prevent="scrollToElement('index')" class="index-link"
      ><p>&#8593; index</p></a
    >

    <h3 id="app-analyst">Analyst</h3>
    <p>
      When you request a financial analysis, all added periods will be taken
      into account. If you have added several periods it will focus on the
      potential evolution of the company, while if you have only added one or
      two periods it will perform a more static analysis of the company.
    </p>
    <p class="important">
      Each analysis will overwrite any previous analysis for that company.
      Likewise, if you have used it previously for a company and add a new
      period, your previous analysis will be deleted, since new information will
      be available to perform a new analysis.
    </p>
    <a @click.prevent="scrollToElement('index')" class="index-link"
      ><p>&#8593; index</p></a
    >

    <h3 id="app-errors">Common Errors</h3>
    <p>
      Given the inconsistency of financial reports and the overwhelming amount
      of formats used, the AI can be led to errors. Some of the most common
      errors are:
    </p>
    <p>
      ❌
      <b
        >Requesting quarterly information when the report shows data for the
        previous 9 months</b
      >. For example, Apple occasionally reports '9 months ended'. Keep in mind
      that certain financial statements report the cumulative total across
      quarters. Exercise extreme care in these scenarios and maintain
      consistency in the files sent. In other cases, for example Spotify, they
      report their most recent quarterly report and in the balance sheet compare
      the data with the previous year-end close, so the balance sheet may show
      2024-Q1 compared to the situation in 2023-Q4.
    </p>
    <p>
      ❌ <b>Sending files that do not contain text</b>, but where each page is
      an image. For example sending a PDF consisting of images. In these cases
      the information cannot be extracted, since our AI is a language model
      (LLM) and will not process any images.
    </p>
    <p>
      ❌
      <b
        >Calculating and interpreting the score and speculated values with
        quarterly or semi-annual data</b
      >. Since they do not reflect the company's annual results.
    </p>
    <p>
      ❌ <b>Using the wrong currency</b>. For example marking the wrong currency
      of the report or sending a report in euros for a company that trades in
      pounds and speculating with its price in euros. Make sure the selected
      currency is correct in the report and that it is the currency you want to
      use to speculate with the stock price. Always check: currency in which the
      stock trades and the currency in which the report is.
    </p>
    <p>
      ❌
      <b>Sending financial reports that contain multiple periods</b>. For
      example a report that reports an income statement showing quarterly,
      semi-annual and annual data, covering 3 years of data and where
      consolidated and non-consolidated versions appear, would contain 18
      potential values for each metric. In these cases the data can get mixed
      up, using data from the wrong period. It is recommended to send files that
      do not cover multiple periods, the more precision in the period, the
      better. Remember that you can select a page range on which you want to
      focus the analysis.
    </p>
    <p>
      ❌
      <b>Entering a wrong period that does not correspond to the report</b>.
    </p>
    <p>
      ❌ <b>Confusing the calendar year with the fiscal year</b>. For example, a
      company may report its 2024 financial statement in March 2024, containing
      most of the data from 2023. Try to always use the same reports for that
      company, for example by sending their annual reports.
    </p>
    <p>
      As a general recommendation, try to always send the same type of report
      for each company and preferably consolidated reports. The more
      standardized and professional the report, the easier it will be to
      correctly extract the information.
    </p>
    <p class="comment">
      Our AI tries to be as accurate as possible, but it is not infallible. If
      you detect any error, remember that you can edit and manually modify any
      data.
    </p>
    <a @click.prevent="scrollToElement('index')" class="index-link"
      ><p>&#8593; index</p></a
    >

    <h3 id="app-limits">Limits</h3>
    <p>
      Why is its use limited? Such capable AI models require a lot of
      computational power, especially when processing large files or performing
      intensive tasks. We establish these limits to ensure that our platform can
      be available to the maximum number of people and all users can integrate
      nodo.finance into their daily workflows. We are working to eliminate these
      limits in the future.
    </p>
    <p>
      Your consumption is mainly based on the file or URL provided, evaluating
      size, length, complexity and computation used both to analyze the text and
      to generate the response. With each request, each user consumes a series
      of 'tokens' that are registered in their account. If you reach the limit,
      you will not be able to use the AI until your next billing period, when
      your consumption will be reset.
    </p>
    <p>
      Financial analyses also consume tokens, depending on the number of periods
      you have added for your company, the analysis will require more or less
      computation, influencing the tokens consumed.
    </p>
    <p>
      You can expect to analyze between 600 and 800 financial reports per month,
      not counting the use of the analyst. If you use the analyst, to minimize
      your consumption, try to add all desired periods before requesting an
      analysis. Remember that these consumptions depend on multiple factors, if
      you send very long and complex reports, your consumption will be higher.
    </p>
    <p class="comment">
      In this early development phase, we are working to optimize our
      infrastructure according to the real needs of the service. To prevent
      abuse and guarantee a stable experience for all users, the number of
      requests per minute and per hour are limited. These restrictions allow us
      to maintain service quality while scaling our capabilities in a controlled
      manner. We ask that you be patient while we improve our computational
      capacity to offer you better service in the future.
    </p>

    <a @click.prevent="scrollToElement('index')" class="index-link"
      ><p>&#8593; index</p></a
    >
  </div>
</template>

<style scoped>
.wrapper {
  width: 780px;
  margin: 0 auto;
  padding: 20px 40px;
  text-align: left;
  background-color: var(--gray-two);
  border-radius: 50px;
  box-shadow: 0 0 30px var(--gray-two);
  box-sizing: border-box;
  min-height: 100vh;
}

.scroll {
  cursor: pointer;
}

pre {
  background-color: var(--gray-three);
  padding: 8px 12px;
  border-radius: 8px;
  max-width: fit-content;
  overflow-x: scroll;
}
h1 {
  font-size: 48px;
  padding: 0;
  margin: 0;
  margin-bottom: 30px;
  letter-spacing: 1px;
  font-weight: 400;
}
h2 {
  padding-bottom: 3px;
  padding-top: 10px;
  font-size: 30px;
  letter-spacing: 1px;
  font-weight: 400;
  border-bottom: 3px solid black;
}
h3 {
  margin: 35px 0 0 0;
  font-size: 20px;
  letter-spacing: 1px;
  font-weight: 400;
  padding-top: 15px;
}
li {
  list-style: disc;
  text-align: left;
  padding-bottom: 8px;
  line-height: 1.3;
  letter-spacing: 0.5px;
}
.index li {
  font-size: 18px;
  padding: 5px 0;
}
li.subindex {
  list-style: circle;
  margin-left: 20px;
}
p {
  line-height: 1.3;
  margin: 20px 0;
  letter-spacing: 0.5px;
}
a p {
  font-size: 20px;
  letter-spacing: 1px;
  margin: 20px 0;
  font-weight: 500;
}

.video-wrapper {
  width: 100%;
  margin: 20px auto;
}
.index-link p {
  font-size: 16px;
  font-weight: 500;
  color: var(--gray-four);
  cursor: pointer;
}

.comment {
  color: var(--gray-six);
  font-size: 14px;
  padding: 8px;
  border-radius: 8px;
}
.important {
  background-color: #ffffaa;
  letter-spacing: 0.6px;
  font-size: 14px;
  padding: 8px;
  border-radius: 8px;
}

.index {
  padding: 0px 20px 30px 20px;
  font-size: 30px;
  letter-spacing: 1px;
  font-weight: 400;
  border-bottom: 3px solid black;
}
.index h2 {
  padding-bottom: 3px;
  padding-top: 10px;
  margin-top: 0;
}

@media (max-width: 880px) {
  .wrapper {
    width: 95%;
    padding: 20px;
    border-radius: 20px;
  }
}
@media (max-width: 420px) {
  .wrapper {
    width: 100%;
    padding: 15px;
    border-radius: 0;
    box-shadow: none;
  }
  h1 {
    font-size: 32px;
  }
}
</style>
