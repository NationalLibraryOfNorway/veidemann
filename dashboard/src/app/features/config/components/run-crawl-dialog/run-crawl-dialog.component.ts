import { Component, ChangeDetectionStrategy, inject } from '@angular/core';
import {MAT_DIALOG_DATA, MatDialogModule, MatDialogRef} from '@angular/material/dialog';
import {RunCrawlReply, RunCrawlRequest} from '../../../../shared/models/controller/controller.model';
import {ConfigObject, Kind} from '../../../../shared/models/config';
import {MatFormFieldModule} from '@angular/material/form-field';
import {MatSelectModule} from '@angular/material/select';
import {FormsModule} from '@angular/forms';
import {MatButtonModule} from '@angular/material/button';

export interface RunCrawlDialogData {
  runCrawlReply: RunCrawlReply;
  configObject: ConfigObject;
  crawlJobs: ConfigObject[];
  numberOfSeeds?: number;
  jobRefId?: string;
}

@Component({
  selector: 'app-run-crawl-dialog',
  templateUrl: './run-crawl-dialog.component.html',
  styleUrls: ['./run-crawl-dialog.component.css'],
  imports: [
    MatButtonModule,
    MatDialogModule,
    MatFormFieldModule,
    MatSelectModule,
    FormsModule,
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true
})

export class RunCrawlDialogComponent {
  data = inject<RunCrawlDialogData>(MAT_DIALOG_DATA);
  private dialogRef = inject<MatDialogRef<RunCrawlDialogComponent>>(MatDialogRef);

  readonly Kind = Kind;

  runCrawlReply: RunCrawlReply;
  configObject: ConfigObject;
  crawlJobs: ConfigObject[];
  numberOfSeeds: number;
  jobRefId: string;

  constructor() {
    const data = this.data;

    this.runCrawlReply = data.runCrawlReply;
    this.configObject = data.configObject;
    this.crawlJobs = data.crawlJobs;
    this.jobRefId = data.jobRefId;
    if (this.configObject.kind === Kind.SEED) {
      this.numberOfSeeds = data.numberOfSeeds ? data.numberOfSeeds : 1;
    }
  }

  get kind(): Kind {
    return this.configObject.kind;
  }

  onRunCrawl() {
    const runCrawlRequest = new RunCrawlRequest();
    let crawlMultiple = false;
    if (this.kind === Kind.SEED) {
      runCrawlRequest.seedId = this.configObject.id;
      runCrawlRequest.jobId = this.jobRefId;
      if (this.numberOfSeeds > 1) {
        crawlMultiple = true;
      }
    } else if (this.kind === Kind.CRAWLJOB) {
      runCrawlRequest.jobId = this.configObject.id;
    }
    this.dialogRef.close({runCrawlRequest, crawlMultiple});
  }
}
