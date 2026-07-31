import { useEffect, useState } from "react";
import type { FormEvent } from "react";
import type { SSHKey } from "../types";
import type { SSHKeyGenerateState, SSHKeyImportState } from "../lib/app-types";
import { useI18n } from "../lib/i18n";
import EmptyState from "../components/EmptyState";
import Modal from "../components/Modal";
import Pagination from "../components/Pagination";

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

function CopyIcon() {
  return (
    <svg className="copy-button-icon" viewBox="0 0 24 24" aria-hidden="true" focusable="false">
      <rect x="9" y="9" width="11" height="11" rx="2.5" />
      <path d="M5.5 15H4.75A1.75 1.75 0 0 1 3 13.25V4.75A1.75 1.75 0 0 1 4.75 3H13.25A1.75 1.75 0 0 1 15 4.75V5.5" />
    </svg>
  );
}

function CheckIcon() {
  return (
    <svg className="copy-button-icon" viewBox="0 0 24 24" aria-hidden="true" focusable="false">
      <path d="M5 12.5L10 17.5L19 7.5" />
    </svg>
  );
}

async function copyText(value: string) {
  if (navigator.clipboard?.writeText) {
    await navigator.clipboard.writeText(value);
    return;
  }

  const textarea = document.createElement("textarea");
  textarea.value = value;
  textarea.setAttribute("readonly", "true");
  textarea.style.position = "fixed";
  textarea.style.left = "-9999px";
  document.body.appendChild(textarea);
  textarea.select();

  try {
    document.execCommand("copy");
  } finally {
    document.body.removeChild(textarea);
  }
}

export default function SSHKeysPage(props: SSHKeysPageProps) {
  const { t } = useI18n();
  const [activeModal, setActiveModal] = useState<SSHKeyModal | null>(null);
  const [copiedSSHKeyID, setCopiedSSHKeyID] = useState<number | null>(null);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const totalPages = Math.max(1, Math.ceil(props.sshKeys.length / pageSize));
  const visibleSSHKeys = props.sshKeys.slice((page - 1) * pageSize, page * pageSize);

  useEffect(() => {
    setPage((current) => Math.min(current, totalPages));
  }, [totalPages]);

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

  function handlePageSizeChange(nextPageSize: number) {
    setPageSize(nextPageSize);
    setPage(1);
  }

  async function handleCopyPublicKey(sshKey: SSHKey) {
    await copyText(sshKey.public_key);
    setCopiedSSHKeyID(sshKey.id);
    window.setTimeout(() => {
      setCopiedSSHKeyID((current) => (current === sshKey.id ? null : current));
    }, 1600);
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
          <>
            <div className="table-wrapper">
              <table className="ssh-key-table">
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
                  {visibleSSHKeys.map((sshKey) => (
                    <tr key={sshKey.id}>
                      <td className="ssh-key-name-cell">
                        <strong className="ssh-key-name">{sshKey.name}</strong>
                        <span className="ssh-key-source">{sshKey.source_type}</span>
                      </td>
                      <td>{sshKey.key_type}</td>
                      <td>
                        <code className="fingerprint-text" title={sshKey.fingerprint}>{sshKey.fingerprint}</code>
                      </td>
                      <td>
                        <div className="public-key-box">
                          <code className="public-key-text" title={sshKey.public_key}>{sshKey.public_key}</code>
                          <button
                            className={`copy-public-key-button${copiedSSHKeyID === sshKey.id ? " copied" : ""}`}
                            aria-label={copiedSSHKeyID === sshKey.id ? t("sshKeysCopied") : t("sshKeysCopyPublicKey")}
                            onClick={() => void handleCopyPublicKey(sshKey)}
                            title={copiedSSHKeyID === sshKey.id ? t("sshKeysCopied") : t("sshKeysCopyPublicKey")}
                            type="button"
                          >
                            {copiedSSHKeyID === sshKey.id ? <CheckIcon /> : <CopyIcon />}
                          </button>
                        </div>
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
            <Pagination
              page={page}
              totalPages={totalPages}
              total={props.sshKeys.length}
              pageSize={pageSize}
              onPageChange={setPage}
              onPageSizeChange={handlePageSizeChange}
            />
          </>
        )}
      </section>

      {activeModal ? (
        <Modal
          title={activeModal === "import" ? t("sshKeysImportTitle") : t("sshKeysGenerateTitle")}
          onClose={closeModal}
        >
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
        </Modal>
      ) : null}
    </div>
  );
}
