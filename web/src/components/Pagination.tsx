import { useI18n } from "../lib/i18n";
import PageSizeSelect from "./PageSizeSelect";

type PaginationProps = {
  page: number;
  totalPages: number;
  total: number;
  pageSize: number;
  disabled?: boolean;
  onPageChange: (page: number) => void;
  onPageSizeChange: (pageSize: number) => void;
};

export default function Pagination(props: PaginationProps) {
  const { t } = useI18n();

  return (
    <div className="pagination-row">
      <div className="pagination-meta">
        <span className="card-meta">
          {t("samplesPageInfo", { page: props.page, totalPages: props.totalPages, total: props.total })}
        </span>
        <PageSizeSelect value={props.pageSize} onChange={props.onPageSizeChange} />
      </div>
      <div className="action-row">
        <button
          className="secondary-button"
          disabled={props.disabled || props.page <= 1}
          onClick={() => props.onPageChange(props.page - 1)}
          type="button"
        >
          {t("previousPage")}
        </button>
        <button
          className="secondary-button"
          disabled={props.disabled || props.page >= props.totalPages}
          onClick={() => props.onPageChange(props.page + 1)}
          type="button"
        >
          {t("nextPage")}
        </button>
      </div>
    </div>
  );
}
