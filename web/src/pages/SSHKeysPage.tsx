import type { FormEvent } from "react";
import type { SSHKey } from "../types";
import type { SSHKeyGenerateState, SSHKeyImportState } from "../lib/app-types";
import { useI18n } from "../lib/i18n";
import EmptyState from "../components/EmptyState";

type SSHKeysPageProps = {
  busy: boolean;
  sshKeys: SSHKey[];
  sshImportForm: SSHKeyImportState;
  sshGenerateForm: SSHKeyGenerateState;
  renamingSSHKeyID: number | null;
  sshRenameName: string;
  setSSHImportForm: React.Dispatch<React.SetStateAction<SSHKeyImportState>>;
  setSSHGenerateForm: React.Dispatch<React.SetStateAction<SSHKeyGenerateState>>;
  onImportSubmit: (event: FormEvent<HTMLFormElement>) => void | Promise<void>;
  onGenerateSubmit: (event: FormEvent<HTMLFormElement>) => void | Promise<void>;
  onDeleteSSHKey: (id: number) => void | Promise<void>;
  onStartRenameSSHKey: (sshKey: SSHKey) => void;
  onCancelRenameSSHKey: () => void;
  setSSHRenameName: React.Dispatch<React.SetStateAction<string>>;
  onRenameSSHKey: (sshKeyID: number) => void | Promise<void>;
};

export default function SSHKeysPage(props: SSHKeysPageProps) {
  const { t } = useI18n();

  return (
    <div className="page-stack">
      <section className="summary-strip">
        <div className="summary-tile teal compact">
          <span>{t("overviewSSHKeysLabel")}</span>
          <strong>{props.sshKeys.length}</strong>
        </div>
        <div className="summary-tile slate compact">
          <span>{t("sshKeysGenerateTitle")}</span>
          <strong>RSA 4096</strong>
        </div>
      </section>

      <div className="grid dashboard-columns">
        <div className="stack-column">
          <section className="panel section-panel">
            <div className="section-intro">
              <div>
                <p className="section-kicker">{t("sshKeysImportTitle")}</p>
                <h3 className="panel-title">{t("sshKeysImportTitle")}</h3>
              </div>
              <p className="section-description">{t("sshKeysPageDescription")}</p>
            </div>
            <form className="form-grid" onSubmit={props.onImportSubmit}>
              <label className="field">
                <span>{t("sshKeysName")}</span>
                <input
                  value={props.sshImportForm.name}
                  onChange={(event) =>
                    props.setSSHImportForm((current) => ({
                      ...current,
                      name: event.target.value,
                    }))
                  }
                  placeholder={t("sshKeysImportPlaceholder")}
                />
              </label>
              <label className="field">
                <span>{t("sshKeysPrivateKey")}</span>
                <textarea
                  rows={10}
                  value={props.sshImportForm.privateKey}
                  onChange={(event) =>
                    props.setSSHImportForm((current) => ({
                      ...current,
                      privateKey: event.target.value,
                    }))
                  }
                  placeholder={t("sshKeysPrivateKeyPlaceholder")}
                />
              </label>
              <button
                className="primary-button"
                disabled={props.busy || !props.sshImportForm.name.trim() || !props.sshImportForm.privateKey.trim()}
                type="submit"
              >
                {t("sshKeysImport")}
              </button>
            </form>
          </section>

          <section className="panel section-panel">
            <div className="section-intro">
              <div>
                <p className="section-kicker">{t("sshKeysGenerateTitle")}</p>
                <h3 className="panel-title">{t("sshKeysGenerateTitle")}</h3>
              </div>
            </div>
            <form className="form-grid" onSubmit={props.onGenerateSubmit}>
              <label className="field">
                <span>{t("sshKeysName")}</span>
                <input
                  value={props.sshGenerateForm.name}
                  onChange={(event) => props.setSSHGenerateForm({ name: event.target.value })}
                  placeholder={t("sshKeysGeneratePlaceholder")}
                />
              </label>
              <button className="primary-button" disabled={props.busy} type="submit">
                {t("sshKeysGenerate")}
              </button>
            </form>
          </section>
        </div>

        <section className="panel section-panel">
          <div className="section-intro">
            <div>
              <p className="section-kicker">{t("sshKeysList")}</p>
              <h3 className="panel-title">{t("sshKeysList")}</h3>
            </div>
            <p className="section-description">{t("sshKeysPageDescription")}</p>
          </div>
          {props.sshKeys.length === 0 ? (
            <EmptyState title={t("sshKeysEmptyTitle")} description={t("sshKeysEmptyDescription")} />
          ) : (
            <div className="list-block card-list spacious">
              {props.sshKeys.map((sshKey) => (
                <article className="card key-card" key={sshKey.id}>
                  <div className="card-header">
                    <div className="stacked-copy">
                      <strong>{sshKey.name}</strong>
                      <span className="card-tag">{sshKey.key_type}</span>
                    </div>
                    <div className="action-row">
                      <button
                        className="secondary-button"
                        onClick={() => props.onStartRenameSSHKey(sshKey)}
                        type="button"
                      >
                        {t("sshKeysRename")}
                      </button>
                      <button className="danger-button" onClick={() => void props.onDeleteSSHKey(sshKey.id)} type="button">
                        {t("sshKeysDelete")}
                      </button>
                    </div>
                  </div>
                  {props.renamingSSHKeyID === sshKey.id ? (
                    <form
                      className="form-grid"
                      onSubmit={(event) => {
                        event.preventDefault();
                        void props.onRenameSSHKey(sshKey.id);
                      }}
                    >
                      <label className="field">
                        <span>{t("sshKeysNewName")}</span>
                        <input
                          value={props.sshRenameName}
                          onChange={(event) => props.setSSHRenameName(event.target.value)}
                          placeholder={t("sshKeysNewNamePlaceholder")}
                        />
                      </label>
                      <div className="action-row">
                        <button className="primary-button" disabled={props.busy || !props.sshRenameName.trim()} type="submit">
                          {t("sshKeysSaveName")}
                        </button>
                        <button className="secondary-button" onClick={props.onCancelRenameSSHKey} type="button">
                          {t("cancel")}
                        </button>
                      </div>
                    </form>
                  ) : null}
                  <p className="card-meta">{t("sshKeysTypeAndSource", { type: sshKey.key_type, source: sshKey.source_type })}</p>
                  <p className="card-meta">{t("sshKeysFingerprint", { fingerprint: sshKey.fingerprint })}</p>
                  <pre className="code-block">{sshKey.public_key}</pre>
                </article>
              ))}
            </div>
          )}
        </section>
      </div>
    </div>
  );
}
