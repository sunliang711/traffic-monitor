import { useState } from "react";
import type { FormEvent } from "react";
import type { NotificationChannel } from "../types";
import type { TelegramFormState, TelegramPreviewState, WebhookFormState, WebhookPreviewState } from "../lib/app-types";
import { useI18n } from "../lib/i18n";

type NotificationsPageProps = {
  busy: boolean;
  notificationChannels: NotificationChannel[];
  webhookForm: WebhookFormState;
  webhookSaved: boolean;
  webhookPreview: WebhookPreviewState | null;
  telegramForm: TelegramFormState;
  telegramSaved: boolean;
  telegramPreview: TelegramPreviewState | null;
  onWebhookFormChange: (updater: (current: WebhookFormState) => WebhookFormState) => void;
  onTelegramFormChange: (updater: (current: TelegramFormState) => TelegramFormState) => void;
  onSaveWebhook: (event: FormEvent<HTMLFormElement>) => void;
  onTestWebhook: () => void;
  onTestTelegram: () => void;
  onSaveTelegram: (event: FormEvent<HTMLFormElement>) => void;
};

export default function NotificationsPage(props: NotificationsPageProps) {
  const { t } = useI18n();
  const [isWebhookConfigOpen, setWebhookConfigOpen] = useState(false);
  const notificationVariables = [
    "{{machine_id}}",
    "{{machine_name}}",
    "{{machine_host}}",
    "{{period_type}}",
    "{{metric_type}}",
    "{{bucket_time}}",
    "{{threshold_mb}}",
    "{{threshold_human_readable}}",
    "{{actual_mb}}",
    "{{actual_human_readable}}",
    "{{alert_key}}",
  ];

  return (
    <div className="page-stack">
      <section className="notification-status-grid" aria-label={t("notificationsChannelStatus")}>
        {props.notificationChannels.map((channel) => (
          <article className="card notification-status-card" key={channel.channel_type}>
            <div className="card-header">
              <div>
                <p className="section-kicker">{t("notificationsChannelStatus")}</p>
                <strong>{channel.channel_type}</strong>
              </div>
              <span className={`status-badge ${channel.enabled ? "ok" : "idle"}`}>
                {channel.enabled ? t("statusEnabled") : t("statusDisabled")}
              </span>
            </div>
            <p className="card-meta">
              {t("notificationsConfigured", { value: channel.configured ? t("statusConfiguredYes") : t("statusConfiguredNo") })}
            </p>
            {channel.url ? <p className="card-meta">{t("notificationsURL", { url: channel.url })}</p> : null}
            {channel.chat_id ? <p className="card-meta">{t("notificationsChatID", { value: channel.chat_id })}</p> : null}
            {channel.token_masked ? <p className="card-meta">{t("notificationsToken", { value: channel.token_masked })}</p> : null}
          </article>
        ))}
      </section>

      <section className="panel section-panel">
        <div className="section-toolbar">
          <div className="section-intro">
            <div>
              <h3 className="panel-title">{t("notificationsWebhookTitle")}</h3>
            </div>
            <p className="section-description">{t("notificationsPageDescription")}</p>
          </div>
          <div className="action-row">
            <button className="secondary-button" onClick={() => setWebhookConfigOpen((current) => !current)} type="button">
              {isWebhookConfigOpen ? t("notificationsCollapseConfig") : t("notificationsExpandConfig")}
            </button>
            <button className="secondary-button" disabled={props.busy} onClick={props.onTestWebhook} type="button">
              {t("notificationsTestWebhook")}
            </button>
            <button
              className="primary-button"
              disabled={props.busy || props.webhookSaved}
              form="webhook-form"
              type="submit"
            >
              {t("notificationsSaveWebhook")}
            </button>
          </div>
        </div>
        <form id="webhook-form" className="form-grid webhook-form-grid" onSubmit={props.onSaveWebhook}>
          <label className="field checkbox-field full-width">
            <input
              checked={props.webhookForm.enabled}
              onChange={(event) => {
                props.onWebhookFormChange((current) => ({ ...current, enabled: event.target.checked }));
              }}
              type="checkbox"
            />
            <span>{t("notificationsEnableWebhook")}</span>
          </label>

          {isWebhookConfigOpen ? (
            <>
              <label className="field">
                <span>{t("notificationsMethod")}</span>
                <select
                  value={props.webhookForm.method}
                  onChange={(event) => {
                    props.onWebhookFormChange((current) => ({
                      ...current,
                      method: event.target.value as "GET" | "POST",
                    }));
                  }}
                >
                  <option value="POST">POST</option>
                  <option value="GET">GET</option>
                </select>
              </label>
              <label className="field">
                <span>URL</span>
                <input
                  value={props.webhookForm.url}
                  onChange={(event) => {
                    props.onWebhookFormChange((current) => ({ ...current, url: event.target.value }));
                  }}
                  placeholder="https://example.com/hook"
                />
              </label>
              <label className="field full-width">
                <span>{t("notificationsHeaders")}</span>
                <textarea
                  rows={4}
                  value={props.webhookForm.headersText}
                  onChange={(event) => {
                    props.onWebhookFormChange((current) => ({ ...current, headersText: event.target.value }));
                  }}
                  placeholder={`{
  "Authorization": "Bearer {{alert_key}}",
  "X-Metric": "{{metric_type}}",
  "X-Machine-Name": "{{machine_name}}"
}`}
                />
              </label>
              <label className="field full-width">
                <span>{t("notificationsBody")}</span>
                <textarea
                  rows={7}
                  value={props.webhookForm.bodyText}
                  onChange={(event) => {
                    props.onWebhookFormChange((current) => ({ ...current, bodyText: event.target.value }));
                  }}
                  placeholder={`{
  "machine_id": "{{machine_id}}",
  "machine_name": "{{machine_name}}",
  "machine_host": "{{machine_host}}",
  "period_type": "{{period_type}}",
  "metric_type": "{{metric_type}}",
  "bucket_time": "{{bucket_time}}",
  "threshold_mb": "{{threshold_mb}}",
  "threshold_human_readable": "{{threshold_human_readable}}",
  "actual_mb": "{{actual_mb}}",
  "actual_human_readable": "{{actual_human_readable}}",
  "alert_key": "{{alert_key}}"
}`}
                />
              </label>
              <div className="card">
                <div className="card-header">
                  <strong>{t("notificationsVariablesTitle")}</strong>
                </div>
                <p className="card-meta">
                  {t("notificationsVariablesDesc")}
                </p>
                <div className="variable-chip-list">
                  {notificationVariables.map((variable) => (
                    <code key={variable}>{variable}</code>
                  ))}
                </div>
              </div>
              {props.webhookPreview ? (
                <div className="card">
                  <div className="card-header">
                    <strong>{t("notificationsPreviewTitle")}</strong>
                  </div>
                  <p className="card-meta">{t("notificationsPreviewURL")}</p>
                  <pre className="code-block">{props.webhookPreview.url || "-"}</pre>
                  <p className="card-meta">{t("notificationsPreviewHeaders")}</p>
                  <pre className="code-block">{props.webhookPreview.headersText || "{}"}</pre>
                  <p className="card-meta">{t("notificationsPreviewBody")}</p>
                  <pre className="code-block">{props.webhookPreview.bodyText || "-"}</pre>
                </div>
              ) : null}
            </>
          ) : (
            <div className="card webhook-collapsed-note full-width">
              <p className="card-meta">{t("notificationsWebhookCollapsed")}</p>
              {props.webhookForm.url ? <p className="card-meta">{t("notificationsURL", { url: props.webhookForm.url })}</p> : null}
            </div>
          )}
        </form>
      </section>

      <section className="panel section-panel telegram-panel">
        <div className="section-toolbar">
          <div className="section-intro">
            <div>
              <h3 className="panel-title">{t("notificationsTelegramTitle")}</h3>
            </div>
            <p className="section-description">{t("notificationsTelegramDescription")}</p>
          </div>
          <div className="action-row">
            <button className="secondary-button" disabled={props.busy} onClick={props.onTestTelegram} type="button">
              {t("notificationsTestTelegram")}
            </button>
            <button
              className="primary-button"
              disabled={props.busy || props.telegramSaved}
              form="telegram-form"
              type="submit"
            >
              {t("notificationsSaveTelegram")}
            </button>
          </div>
        </div>
        <form id="telegram-form" className="form-grid telegram-form-grid" onSubmit={props.onSaveTelegram}>
          <label className="field checkbox-field full-width">
            <input
              checked={props.telegramForm.enabled}
              onChange={(event) => {
                props.onTelegramFormChange((current) => ({ ...current, enabled: event.target.checked }));
              }}
              type="checkbox"
            />
            <span>{t("notificationsEnableTelegram")}</span>
          </label>

          {props.telegramForm.enabled ? (
            <>
              <label className="field">
                <span>Bot Token</span>
                <input
                  value={props.telegramForm.botToken}
                  onChange={(event) => {
                    props.onTelegramFormChange((current) => ({ ...current, botToken: event.target.value }));
                  }}
                  placeholder={t("notificationsBotTokenPlaceholder")}
                />
              </label>
              <label className="field">
                <span>Chat ID</span>
                <input
                  value={props.telegramForm.chatID}
                  onChange={(event) => {
                    props.onTelegramFormChange((current) => ({ ...current, chatID: event.target.value }));
                  }}
                />
              </label>
              <label className="field full-width">
                <span>{t("notificationsTelegramMessage")}</span>
                <textarea
                  rows={5}
                  value={props.telegramForm.messageText}
                  onChange={(event) => {
                    props.onTelegramFormChange((current) => ({ ...current, messageText: event.target.value }));
                  }}
                  placeholder="machine={{machine_name}} actual={{actual_human_readable}}"
                />
              </label>
              <div className="card">
                <div className="card-header">
                  <strong>{t("notificationsVariablesTitle")}</strong>
                </div>
                <p className="card-meta">
                  {t("notificationsTelegramVariablesDesc")}
                </p>
                <div className="variable-chip-list">
                  {notificationVariables.map((variable) => (
                    <code key={variable}>{variable}</code>
                  ))}
                </div>
              </div>
              {props.telegramPreview ? (
                <div className="card">
                  <div className="card-header">
                    <strong>{t("notificationsPreviewTitle")}</strong>
                  </div>
                  <p className="card-meta">{t("notificationsPreviewMessage")}</p>
                  <pre className="code-block">{props.telegramPreview.messageText || "-"}</pre>
                </div>
              ) : null}
            </>
          ) : (
            <div className="card telegram-collapsed-note full-width">
              <p className="card-meta">{t("notificationsTelegramCollapsed")}</p>
            </div>
          )}
        </form>
      </section>
    </div>
  );
}
