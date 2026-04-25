import type { FormEvent } from "react";
import type { SSHKey } from "../types";
import type { SSHKeyGenerateState, SSHKeyImportState } from "../lib/app-types";

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
  return (
    <div className="grid two-columns">
      <section className="panel">
        <h3 className="panel-title">导入已有 SSH Key</h3>
        <form className="form-grid" onSubmit={props.onImportSubmit}>
          <label className="field">
            <span>名称</span>
            <input
              value={props.sshImportForm.name}
              onChange={(event) =>
                props.setSSHImportForm((current) => ({
                  ...current,
                  name: event.target.value,
                }))
              }
              placeholder="例如：prod-root"
            />
          </label>
          <label className="field">
            <span>私钥</span>
            <textarea
              rows={10}
              value={props.sshImportForm.privateKey}
              onChange={(event) =>
                props.setSSHImportForm((current) => ({
                  ...current,
                  privateKey: event.target.value,
                }))
              }
              placeholder="粘贴 OpenSSH 私钥"
            />
          </label>
          <button
            className="primary-button"
            disabled={props.busy || !props.sshImportForm.name.trim() || !props.sshImportForm.privateKey.trim()}
            type="submit"
          >
            导入
          </button>
        </form>
      </section>

      <section className="panel">
        <h3 className="panel-title">生成新 Keypair</h3>
        <form className="form-grid" onSubmit={props.onGenerateSubmit}>
          <label className="field">
            <span>名称</span>
            <input
              value={props.sshGenerateForm.name}
              onChange={(event) => props.setSSHGenerateForm({ name: event.target.value })}
              placeholder="例如：ops-generated"
            />
          </label>
          <button className="primary-button" disabled={props.busy} type="submit">
            生成
          </button>
        </form>

        <div className="list-block">
          <h4>SSH Key 列表</h4>
          {props.sshKeys.map((sshKey) => (
            <article className="card" key={sshKey.id}>
              <div className="card-header">
                <strong>{sshKey.name}</strong>
                <div className="action-row">
                  <button
                    className="secondary-button"
                    onClick={() => props.onStartRenameSSHKey(sshKey)}
                    type="button"
                  >
                    重命名
                  </button>
                  <button className="danger-button" onClick={() => void props.onDeleteSSHKey(sshKey.id)} type="button">
                    删除
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
                    <span>新名称</span>
                    <input
                      value={props.sshRenameName}
                      onChange={(event) => props.setSSHRenameName(event.target.value)}
                      placeholder="请输入新的 SSH Key 名称"
                    />
                  </label>
                  <div className="action-row">
                    <button className="primary-button" disabled={props.busy || !props.sshRenameName.trim()} type="submit">
                      保存名称
                    </button>
                    <button className="secondary-button" onClick={props.onCancelRenameSSHKey} type="button">
                      取消
                    </button>
                  </div>
                </form>
              ) : null}
              <p className="card-meta">
                类型：{sshKey.key_type} / 来源：{sshKey.source_type}
              </p>
              <p className="card-meta">指纹：{sshKey.fingerprint}</p>
              <pre className="code-block">{sshKey.public_key}</pre>
            </article>
          ))}
        </div>
      </section>
    </div>
  );
}
