import { ChangeDetectionStrategy,Component,inject,OnInit } from '@angular/core';
import { ReactiveFormsModule } from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MAT_DIALOG_DATA,MatDialogModule,MatDialogRef } from '@angular/material/dialog';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatSelectModule } from '@angular/material/select';
import { MatSlideToggleModule } from '@angular/material/slide-toggle';
import { FlexDirective } from '@ngbracket/ngx-layout';
import { LayoutGapDirective } from '@ngbracket/ngx-layout/flex';
import { CrawlJobDetailsComponent } from '..';
import { ConfigObject } from '../../../../../shared/models/config';
import { ConfigDialogData } from '../../../func';
import { DurationPickerComponent } from '../../durationpicker/duration-picker';
import { FilesizeInputComponent } from '../../filesize-input/filesize-input.component';
import { MetaComponent } from '../../meta/meta.component';

@Component({
  selector: 'app-crawljob-dialog',
  templateUrl: './crawljob-dialog.component.html',
  styleUrls: ['./crawljob-dialog.component.css'],
  imports: [
    FlexDirective,
    LayoutGapDirective,
    MatButtonModule,
    MatDialogModule,
    MatFormFieldModule,
    MatSelectModule,
    MatSlideToggleModule,
    MetaComponent,
    ReactiveFormsModule,
    DurationPickerComponent,
    FilesizeInputComponent
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true
})
export class CrawlJobDialogComponent extends CrawlJobDetailsComponent implements OnInit {
  data = inject<ConfigDialogData>(MAT_DIALOG_DATA);
  dialogRef = inject<MatDialogRef<CrawlJobDialogComponent>>(MatDialogRef);


  constructor() {

    super();

    this.createForm();
    this.configObject = this.data.configObject;
    this.crawlConfigs = this.data.options.crawlConfigs;
    this.crawlScheduleConfigs = this.data.options.crawlScheduleConfigs;
    this.scopeScripts = this.data.options.scopeScripts;
  }

  ngOnInit(): void {
    this.updateForm();
  }

  onDialogClose(): ConfigObject {
    return this.prepareSave();
  }
}
