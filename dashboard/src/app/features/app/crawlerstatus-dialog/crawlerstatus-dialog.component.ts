import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import {MAT_DIALOG_DATA, MatDialogModule} from '@angular/material/dialog';
import {MatButtonModule} from '@angular/material/button';

@Component({
  selector: 'app-crawlerstatus-dialog',
  templateUrl: './crawlerstatus-dialog.component.html',
  styleUrls: ['./crawlerstatus-dialog.component.css'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    MatButtonModule,
    MatDialogModule
  ],
  standalone: true
})
export class CrawlerStatusDialogComponent {
  data = inject(MAT_DIALOG_DATA);

  shouldPause: boolean;

  constructor() {
    const data = this.data;

    this.shouldPause = data.shouldPause;
  }

}
