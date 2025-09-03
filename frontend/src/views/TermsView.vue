<script setup>
import { useHead } from '@unhead/vue'
import { config } from '@config'

useHead({
  title: 'nodo.finance - Terms of Service',
  meta: [
    {
      name: 'twitter:title',
      content: 'nodo.finance - Terms of Service',
    },
    {
      property: 'og:title',
      content: 'nodo.finance - Terms of Service',
    },
    {
      name: 'robots',
      content: 'index, follow',
    },
  ],
})
</script>

<template>
  <div class="wrapper">
    <h1>Terms of Service</h1>
    <ol>
      <li>
        nodo.finance is in an early development phase and, as such, may contain
        errors that affect the experience, functionality and availability of the
        website. Services and content are provided "as is" and "as available",
        without warranties of any kind. nodo.finance will not be responsible for
        any damage, loss or harm of any kind arising directly or indirectly from
        the use or inability to use this website. Users who choose to use
        nodo.finance in this initial phase acknowledge and accept the risk and
        understand that their experience may be affected. In our commitment to
        continuous improvement of our services, we ask that you inform us of any
        errors or problems encountered, as well as any suggestions. Your
        feedback is valuable and contributes significantly to our development
        phases:
        {{ config.contactEmail }}.
      </li>
      <li>
        Artificial intelligence (AI) models present information based on complex
        algorithms and machine learning to provide recommendations, analysis or
        any other type of AI-generated information. While our AI is developed
        and updated with the goal of being as accurate as possible, we cannot
        guarantee the accuracy, relevance, reliability, or applicability of
        AI-generated content. Users should verify information and exercise their
        own critical judgment when interpreting and acting on the information
        provided. nodo.finance is not responsible for decisions made based on
        the use of this information. Files processed may be used by nodo.finance
        to refine AI models and their capabilities.
      </li>
      <li>
        Payments made on this website are processed by Stripe. Stripe complies
        with Payment Card Industry Data Security Standards (PCI DSS).
        nodo.finance does not store any type of financial data. Users must
        ensure that the payment information provided is correct. We are not
        responsible for unprocessed payments due to incorrect information. Only
        one subscription per user is allowed. Make sure to make the payment
        using the email associated with your account. Subscriptions to our
        website are renewed monthly. Users may cancel their subscription at any
        time from their control panel, stopping future automatic renewals. Given
        the digital nature of our services and the immediate access provided to
        content and resources, all payments are final and non-refundable. By
        completing a purchase, you acknowledge and agree that no refunds will be
        issued for already billed subscriptions, regardless of use or access to
        the services provided, except in exceptional circumstances at
        nodo.finance's discretion. This policy applies even in the presence of
        errors or service interruptions. Subscription cancellations must be made
        before the renewal date to avoid charges in the next billing cycle. We
        reserve the right to modify the prices of our subscriptions and the
        services offered without prior notice. Any price changes will be
        communicated to users in advance through our website or by email.
      </li>
      <li>
        nodo.finance uses analytics to collect anonymous data about traffic,
        such as visits and visitor behavior patterns. This information helps us
        improve our website and offer a better user experience. Google Analytics
        uses cookies to track visitor interactions but does not collect personal
        information. You can read more about how Google uses data at
        <a
          style="color: grey"
          target="_blank"
          href="https://policies.google.com/technologies/partner-sites"
          >Google Terms and Privacy</a
        >
        Disabling cookies or using a browser extension to exclude analytics may
        affect your user experience and some functionalities of our website.
      </li>
      <li>
        Unauthorized access, data extraction, scripting techniques, use of bots,
        phishing, identity theft and other illegal or unethical practices will
        result in restricted access, legal action or other appropriate measures.
        Users are responsible for complying with all local laws and regulations
        when accessing and using our services. The use of VPNs, proxies or any
        other obfuscation method may affect the functionality and availability
        of our services.
      </li>
      <li>
        Users agree to defend and hold us harmless from any claims or damages
        related to their use of the website or violation of these terms.
      </li>
      <li>
        nodo.finance reserves the right to limit or cancel access without prior
        notice.
      </li>
      <li>
        These terms are subject to change without prior notice. Periodically
        review the terms of service to stay updated.
      </li>
    </ol>
  </div>
</template>

<style scoped>
.wrapper {
  width: 700px;
  margin: 0 auto;
  padding: 20px 40px;
  text-align: left;
  background-color: var(--gray-two);
  border-radius: 50px;
  box-shadow: 0 0 30px var(--gray-two);
  box-sizing: border-box;
}

h1 {
  font-size: 48px;
  padding: 0;
  margin: 0;
  margin-bottom: 30px;
  letter-spacing: 1px;
  font-weight: 400;
}

ol {
  width: 100%;
  margin: 0 auto;
  padding: 0;
  padding-bottom: 30px;
}
li {
  list-style: decimal;
  text-align: left;
  padding-bottom: 15px;
  line-height: 1.3;
  letter-spacing: 0.5px;
}

@media (max-width: 880px) {
  .wrapper {
    width: 95%;
    padding: 20px 20px 20px 30px;
    border-radius: 20px;
  }
  li {
    font-size: 15px;
  }
}
@media (max-width: 420px) {
  .wrapper {
    width: 100%;
    padding: 15px 15px 15px 30px;
    border-radius: 0;
    box-shadow: none;
  }
  h1 {
    font-size: 32px;
  }
}
</style>
