import { useState } from "react";
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

type SSHKeyModal = "import" | "generate";

export default function SSHKeysPage(props: SSHKeysPageProps) {
  const { t } = useI18n();
  const [activeModal, setActiveModal] = useState<SSHKeyModal | null>(null);

  function closeModal() {
    setActiveModal(null);
  }

  async function handleImportSubmit(event: FormEvent<HTMLFormElement>) {
    await props.onImportSubmit(event);
    setActiveModal(null);
  }

  async function handleGenerateSubmit(event: FormEvent<HTMLFormElement>) {
    await props.onGenerateSubmit(event);
    setActiveModal(null);
  }

  return (
    <div className="page-stack">
      <section className="summary-strip">
        <div className="summary-tile teal compact">
          <span>{t("overviewSSHKeysLabel")}</span>
          <strong>{props.sshKeys.length}</strong>
        </div>
        <div className="summary-tile slate compact">
          <span>{t("sshKeysGenerateTitle")}</span>
          <strong>ED25519</strong>
        </div>
      </section>

      <section className="panel section-panel list-panel">
        <div className="section-toolbar">
          <div className="section-intro">
            <div>
              <p className="section-kicker">{t("sshKeysList")}</p>
              <h3 className="panel-title">{t("sshKeysList")}</h3>
            </div>
            <p className="section-description">{t("sshKeysPageDescription")}</p>
          </div>
          <div className="action-row">
            <button className="secondary-button" onClick={() => setActiveModal("generate")} type="button">
              {t("sshKeysGenerate")}
            </button>
            <button className="primary-button" onClick={() => setActiveModal("import")} type="button">
              {t("sshKeysImport")}
            </button>
          </div>
        </div>

        {props.sshKeys.length === 0 ? (
          <EmptyState
            title={t("sshKeysEmptyTitle")}
            description={t("sshKeysEmptyDescription")}
            action={
              <div className="action-row">
                <button className="secondary-button" onClick={() => setActiveModal("generate")} type="button">
                  {t("sshKeysGenerate")}
                </button>
                <button className="primary-button" onClick={() => setActiveModal("import")} type="button">
                  {t("sshKeysImport")}
                </button>
              </div>
            }
          />
        ) : (
          <div className="table-wrapper">
            <table>
              <thead>
                <tr>
                  <th>{t("sshKeysName")}</th>
                  <th>{t("sshKeysKeyType")}</th>
                  <th>{t("sshKeysFingerprintColumn")}</th>
                  <th>{t("sshKeysPublicKey")}</th>
                  <th>{t("machinesActions")}</th>
                </tr>
              </thead>
              <tbody>
                {props.sshKeys.map((sshKey) => (
                  <tr key={sshKey.id}>
                    <td>
                      <div className="stacked-copy">
                        <strong>{sshKey.name}</strong>
                        <span className="card-tag">{sshKey.source_type}</span>
                      </div>
                    </td>
                    <td>{sshKey.key_type}</td>
                    <td>
                      <span className="table-text-muted">{sshKey.fingerprint}</span>
                    </td>
                    <td>
                      <pre className="code-block table-code-block">{sshKey.public_key}</pre>
                    </td>
                    <td>
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
                      {props.renamingSSHKeyID === sshKey.id ? (
                        <form
                          className="inline-edit-form"
                          onSubmit={(event) => {
                            event.preventDefault();
                            void props.onRenameSSHKey(sshKey.id);
                          }}
                        >
                          <input
                            value={props.sshRenameName}
                            onChange={(event) => props.setSSHRenameName(event.target.value)}
                            placeholder={t("sshKeysNewNamePlaceholder")}
                          />
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
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>

      {activeModal ? (
        <div className="modal-backdrop" role="presentation">
          <section className="modal-panel" aria-modal="true" role="dialog">
            <div className="modal-header">
              <div>
                <p className="section-kicker">
                  {activeModal === "import" ? t("sshKeysImportTitle") : t("sshKeysGenerateTitle")}
                </p>
                <h3 className="panel-title">
                  {activeModal === "import" ? t("sshKeysImportTitle") : t("sshKeysGenerateTitle")}
                </h3>
              </div>
              <button className="secondary-button modal-close-button" onClick={closeModal} type="button">
                {t("cancel")}
              </button>
            </div>

            {activeModal === "import" ? (
              <form className="form-grid" onSubmit={handleImportSubmit}>
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
                <div className="modal-actions">
                  <button className="secondary-button" onClick={closeModal} type="button">
                    {t("cancel")}
                  </button>
                  <button
                    className="primary-button"
                    disabled={props.busy || !props.sshImportForm.name.trim() || !props.sshImportForm.privateKey.trim()}
                    type="submit"
                  >
                    {t("sshKeysImport")}
                  </button>
                </div>
              </form>
            ) : (
              <form className="form-grid" onSubmit={handleGenerateSubmit}>
                <label className="field">
                  <span>{t("sshKeysName")}</span>
                  <input
                    value={props.sshGenerateForm.name}
                    onChange={(event) => props.setSSHGenerateForm({ name: event.target.value })}
                    placeholder={t("sshKeysGeneratePlaceholder")}
                  />
                </label>
                <div className="modal-actions">
                  <button className="secondary-button" onClick={closeModal} type="button">
                    {t("cancel")}
                  </button>
                  <button className="primary-button" disabled={props.busy || !props.sshGenerateForm.name.trim()} type="submit">
                    {t("sshKeysGenerate")}
                  </button>
                </div>
              </form>
            )}
          </section>
        </div>
      ) : null}
    </div>
  );
}
