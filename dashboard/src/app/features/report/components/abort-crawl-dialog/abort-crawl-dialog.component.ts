import {ChangeDetectionStrategy, Component, inject} from '@angular/core';
import {MAT_DIALOG_DATA, MatDialogModule, MatDialogRef} from '@angular/material/dialog';
import {ExecutionId} from '../../../../shared/models/controller/controller.model';
import {CrawlExecutionStatus, JobExecutionStatus} from '../../../../shared/models/report';
import {MatButtonModule} from '@angular/material/button';

interface AbortCrawlDialogData {
  crawlExecutionStatus?: CrawlExecutionStatus;
  jobExecutionStatus?: JobExecutionStatus;
}

@Component({
  selector: 'app-abort-crawl-dialog',
  templateUrl: './abort-crawl-dialog.component.html',
  styleUrls: ['./abort-crawl-dialog.component.css'],
  imports: [
    MatButtonModule,
    MatDialogModule,
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true
})
export class AbortCrawlDialogComponent {
  data = inject<AbortCrawlDialogData>(MAT_DIALOG_DATA);
  private dialogRef = inject<MatDialogRef<AbortCrawlDialogComponent>>(MatDialogRef);

  onAbortCrawl() {
    const id = this.data.jobExecutionStatus?.id || this.data.crawlExecutionStatus?.id;
    if (id) {
      this.dialogRef.close(new ExecutionId({id}));
    }
  }
}
